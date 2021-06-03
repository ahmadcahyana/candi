package rabbitmqworker

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
	"github.com/streadway/amqp"
)

type rabbitmqWorker struct {
	ctx           context.Context
	ctxCancelFunc func()

	ch        *amqp.Channel
	shutdown  chan struct{}
	semaphore []chan struct{}
	wg        sync.WaitGroup
	channels  []reflect.SelectCase
	handlers  map[string]types.WorkerHandlerFunc
}

// NewWorker create new rabbitmq consumer
func NewWorker(service factory.ServiceFactory) factory.AppServerFactory {
	if service.GetDependency().GetBroker().GetConfiguration(types.RabbitMQ) == nil {
		panic("Missing RabbitMQ configuration")
	}

	worker := new(rabbitmqWorker)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())
	worker.ch = service.GetDependency().GetBroker().GetConfiguration(types.RabbitMQ).(*amqp.Channel)

	worker.shutdown = make(chan struct{})
	// add shutdown channel to first index
	worker.channels = append(worker.channels, reflect.SelectCase{
		Dir: reflect.SelectRecv, Chan: reflect.ValueOf(worker.shutdown),
	})
	worker.handlers = make(map[string]types.WorkerHandlerFunc)

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(types.RabbitMQ); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				logger.LogYellow(fmt.Sprintf(`[RABBITMQ-CONSUMER] (queue): %-15s  --> (module): "%s"`, `"`+handler.Pattern+`"`, m.Name()))
				queueChan, err := setupQueueConfig(worker.ch, handler.Pattern)
				if err != nil {
					panic(err)
				}

				worker.channels = append(worker.channels, reflect.SelectCase{
					Dir: reflect.SelectRecv, Chan: reflect.ValueOf(queueChan),
				})
				worker.handlers[handler.Pattern] = handler.HandlerFunc
				worker.semaphore = append(worker.semaphore, make(chan struct{}, 1))
			}
		}
	}

	if len(worker.channels) == 1 {
		log.Println("rabbitmq consumer: no queue provided")
	} else {
		fmt.Printf("\x1b[34;1m⇨ RabbitMQ consumer running with %d queue. Broker: %s\x1b[0m\n\n", len(worker.channels)-1,
			candihelper.MaskingPasswordURL(env.BaseEnv().RabbitMQ.Broker))
	}

	return worker
}

func (r *rabbitmqWorker) Serve() {

	for {
		chosen, value, ok := reflect.Select(r.channels)
		if !ok {
			continue
		}

		// if shutdown channel captured, break loop (no more jobs will run)
		if chosen == 0 {
			return
		}

		// exec handler
		if msg, ok := value.Interface().(amqp.Delivery); ok {
			go r.processMessage(r.semaphore[chosen-1], msg)
		}
	}
}

func (r *rabbitmqWorker) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping RabbitMQ Worker...\x1b[0m")
	defer func() { recover(); log.Println("\x1b[33;1mStopping RabbitMQ Worker:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	r.ctxCancelFunc()
	r.shutdown <- struct{}{}
	var runningJob int
	for _, sem := range r.semaphore {
		runningJob += len(sem)
	}
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mRabbitMQ Worker:\x1b[0m waiting %d job until done...\x1b[0m\n", runningJob)
	}

	r.wg.Wait()
	r.ch.Close()
}

func (r *rabbitmqWorker) Name() string {
	return string(types.RabbitMQ)
}

func (r *rabbitmqWorker) processMessage(semaphore chan struct{}, msg amqp.Delivery) {
	semaphore <- struct{}{}
	r.wg.Add(1)
	go func(message amqp.Delivery) {
		defer func() {
			recover()
			r.wg.Done()
			<-semaphore
		}()

		if r.ctx.Err() != nil {
			logger.LogRed("rabbitmq_consumer > ctx root err: " + r.ctx.Err().Error())
			return
		}

		var err error
		trace, ctx := tracer.StartTraceWithContext(r.ctx, "RabbitMQConsumer")
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}

			if err == nil && !env.BaseEnv().RabbitMQ.AutoACK {
				message.Ack(false)
			}

			trace.SetError(err)
			logger.LogGreen("rabbitmq_consumer > trace_url: " + tracer.GetTraceURL(ctx))
			trace.Finish()
		}()

		trace.SetTag("broker", candihelper.MaskingPasswordURL(env.BaseEnv().RabbitMQ.Broker))
		trace.SetTag("exchange", message.Exchange)
		trace.SetTag("routing_key", message.RoutingKey)
		trace.Log("header", message.Headers)
		trace.Log("body", message.Body)
		err = r.handlers[message.RoutingKey](ctx, message.Body)
	}(msg)
}
