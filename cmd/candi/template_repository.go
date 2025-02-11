package main

const (
	templateRepository = `// {{.Header}} DO NOT EDIT.

package repository

import (
	"sync"

	"{{.LibraryName}}/codebase/factory/dependency"
)

var (
	once sync.Once
)

// SetSharedRepository set the global singleton "RepoSQL" and "RepoMongo" implementation
func SetSharedRepository(deps dependency.Dependency) {
	once.Do(func() {
		{{if not .SQLDeps}}// {{end}}setSharedRepoSQL(deps.GetSQLDatabase().ReadDB(), deps.GetSQLDatabase().WriteDB())
		{{if not .MongoDeps}}// {{end}}setSharedRepoMongo(deps.GetMongoDatabase().ReadDB(), deps.GetMongoDatabase().WriteDB())
	})
}
`

	templateRepositoryUOWSQL = `// {{.Header}}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	// @candi:repositoryImport

	"{{.LibraryName}}/tracer"` +
		`{{if .SQLUseGORM}}
	"gorm.io/driver/{{.SQLDriver}}"
	"gorm.io/gorm"{{end}}` + `
)

type (
	// RepoSQL abstraction
	RepoSQL interface {
		WithTransaction(ctx context.Context, txFunc func(ctx context.Context, repo RepoSQL) error) (err error)
		Free()

		// @candi:repositoryMethod
	}

	repoSQLImpl struct {
		readDB, writeDB *{{if .SQLUseGORM}}gorm{{else}}sql{{end}}.DB` + "{{if not .SQLUseGORM}}\n		tx    *sql.Tx{{end}}" + `
	
		// register all repository from modules
		// @candi:repositoryField
	}
)

var (
	globalRepoSQL RepoSQL
)

// setSharedRepoSQL set the global singleton "RepoSQL" implementation
func setSharedRepoSQL(readDB, writeDB *sql.DB) {
	{{if .SQLUseGORM}}gormRead, err := gorm.Open({{.SQLDriver}}.New({{.SQLDriver}}.Config{
		Conn: readDB,
	}), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	gormWrite, err := gorm.Open({{.SQLDriver}}.New({{.SQLDriver}}.Config{
		Conn: writeDB,
	}), &gorm.Config{SkipDefaultTransaction: true})

	if err != nil {
		panic(err)
	}{{end}}
	globalRepoSQL = NewRepositorySQL({{if .SQLUseGORM}}gormRead, gormWrite{{else}}readDB, writeDB, nil{{end}})
}

// GetSharedRepoSQL returns the global singleton "RepoSQL" implementation
func GetSharedRepoSQL() RepoSQL {
	return globalRepoSQL
}

// NewRepositorySQL constructor
func NewRepositorySQL(readDB, writeDB *{{if .SQLUseGORM}}gorm{{else}}sql{{end}}.DB{{if not .SQLUseGORM}}, tx *sql.Tx{{end}}) RepoSQL {

	return &repoSQLImpl{
		readDB: readDB, writeDB: writeDB,{{if not .SQLUseGORM}} tx: tx,{{end}}

		// @candi:repositoryConstructor
	}
}

// WithTransaction run transaction for each repository with context, include handle canceled or timeout context
func (r *repoSQLImpl) WithTransaction(ctx context.Context, txFunc func(ctx context.Context, repo RepoSQL) error) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "RepoSQL:Transaction")
	defer trace.Finish()

	tx{{if not .SQLUseGORM}}, err{{end}} := r.writeDB.Begin()` + "{{if .SQLUseGORM}}\n	err = tx.Error{{end}}" + `
	if err != nil {
		return err
	}

	// reinit new repository in different memory address with tx value
	manager := NewRepositorySQL(r.readDB, {{if not .SQLUseGORM}}r.writeDB, {{end}}tx)
	defer func() {
		if err != nil {
			tx.Rollback()
			trace.SetError(err)
		} else {
			tx.Commit()
		}
		manager.Free()
	}()

	errChan := make(chan error)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic: %v", r)
			}
			close(errChan)
		}()

		if err := txFunc(ctx, manager); err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("Canceled or timeout: %v", ctx.Err())
	case e := <-errChan:
		return e
	}
}

func (r *repoSQLImpl) Free() {
	// make nil all repository
	// @candi:repositoryDestructor
}

// @candi:repositoryImplementation
`

	templateRepositoryUOWMongo = `// {{.Header}} DO NOT EDIT.

package repository

import (
	"go.mongodb.org/mongo-driver/mongo"

	// @candi:repositoryImport
)

type (
	// RepoMongo abstraction
	RepoMongo interface {
		// @candi:repositoryMethod
	}

	repoMongoImpl struct {
		readDB, writeDB *mongo.Database

		// register all repository from modules
		// @candi:repositoryField
	}
)

var globalRepoMongo RepoMongo

// setSharedRepoMongo set the global singleton "RepoMongo" implementation
func setSharedRepoMongo(readDB, writeDB *mongo.Database) {
	globalRepoMongo = &repoMongoImpl{
		readDB: readDB, writeDB: writeDB,

		// @candi:repositoryConstructor
	}
}

// GetSharedRepoMongo returns the global singleton "RepoMongo" implementation
func GetSharedRepoMongo() RepoMongo {
	return globalRepoMongo
}

// @candi:repositoryImplementation
`

	templateRepositoryAbstraction = `// {{.Header}}

package repository

import (
	"context"

	"{{.LibraryName}}/candishared"
	shareddomain "{{$.PackagePrefix}}/pkg/shared/domain"
)

// {{clean (upper .ModuleName)}}Repository abstract interface
type {{clean (upper .ModuleName)}}Repository interface {
	FetchAll(ctx context.Context, filter *candishared.Filter) ([]shareddomain.{{clean (upper .ModuleName)}}, error)
	Count(ctx context.Context, filter *candishared.Filter) int
	Find(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) error
	Save(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) error
	Delete(ctx context.Context, id string) (err error)
}
`

	templateRepositoryMongoImpl = `// {{.Header}}

package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	shareddomain "{{$.PackagePrefix}}/pkg/shared/domain"

	"{{.LibraryName}}/candihelper"
	"{{.LibraryName}}/candishared"
	"{{.LibraryName}}/tracer"
)

type {{clean .ModuleName}}RepoMongo struct {
	readDB, writeDB *mongo.Database
	collection      string
}

// New{{clean (upper .ModuleName)}}RepoMongo mongo repo constructor
func New{{clean (upper .ModuleName)}}RepoMongo(readDB, writeDB *mongo.Database) {{clean (upper .ModuleName)}}Repository {
	return &{{clean .ModuleName}}RepoMongo{
		readDB, writeDB, "{{clean .ModuleName}}s",
	}
}

func (r *{{clean .ModuleName}}RepoMongo) FetchAll(ctx context.Context, filter *candishared.Filter) (data []shareddomain.{{clean (upper .ModuleName)}}, err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoMongo:FetchAll")
	defer func() { trace.SetError(err); trace.Finish() }()

	where := bson.M{}
	trace.SetTag("query", where)

	findOptions := options.Find()
	if len(filter.OrderBy) > 0 {
		findOptions.SetSort(filter)
	}

	if !filter.ShowAll {
		findOptions.SetLimit(int64(filter.Limit))
		findOptions.SetSkip(int64(filter.Offset))
	}
	cur, err := r.readDB.Collection(r.collection).Find(ctx, where, findOptions)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	cur.All(ctx, &data)
	return
}

func (r *{{clean .ModuleName}}RepoMongo) Find(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoMongo:Find")
	defer func() { trace.SetError(err); trace.Finish() }()

	bsonWhere := make(bson.M)
	if data.ID != "" {
		bsonWhere["_id"] = data.ID
	}
	trace.SetTag("query", bsonWhere)

	return r.readDB.Collection(r.collection).FindOne(ctx, bsonWhere).Decode(data)
}

func (r *{{clean .ModuleName}}RepoMongo) Count(ctx context.Context, filter *candishared.Filter) int {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoMongo:Count")
	defer trace.Finish()

	where := bson.M{}
	count, err := r.readDB.Collection(r.collection).CountDocuments(trace.Context(), where)
	trace.SetError(err)
	return int(count)
}

func (r *{{clean .ModuleName}}RepoMongo) Save(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoMongo:Save")
	defer func() { trace.SetError(err); trace.Finish() }()
	tracer.Log(ctx, "data", data)

	data.ModifiedAt = time.Now()
	if data.ID == "" {
		data.ID = primitive.NewObjectID().Hex()
		data.CreatedAt = time.Now()
		_, err = r.writeDB.Collection(r.collection).InsertOne(ctx, data)
	} else {
		opt := options.UpdateOptions{
			Upsert: candihelper.ToBoolPtr(true),
		}
		_, err = r.writeDB.Collection(r.collection).UpdateOne(ctx,
			bson.M{
				"_id": data.ID,
			},
			bson.M{
				"$set": data,
			}, &opt)
	}

	return
}

func (r *{{clean .ModuleName}}RepoMongo) Delete(ctx context.Context, id string) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoMongo:Save")
	defer func() { trace.SetError(err); trace.Finish() }()

	_, err = r.writeDB.Collection(r.collection).DeleteOne(ctx, bson.M{"_id": id})
	return
}
`

	templateRepositorySQLImpl = `// {{.Header}}

package repository

import (
	"context"` + `{{if not .SQLUseGORM}}
	"database/sql"{{end}}` + `
	"time"

	"{{.LibraryName}}/candishared"
	shareddomain "{{$.PackagePrefix}}/pkg/shared/domain"

	"{{.LibraryName}}/tracer"` +
		`{{if .SQLUseGORM}}
	"gorm.io/gorm"{{end}}` + `
)

type {{clean .ModuleName}}RepoSQL struct {
	readDB, writeDB *{{if .SQLUseGORM}}gorm{{else}}sql{{end}}.DB` + "{{if not .SQLUseGORM}}\n	tx              *sql.Tx{{end}}" + `
}

// New{{clean (upper .ModuleName)}}RepoSQL mongo repo constructor
func New{{clean (upper .ModuleName)}}RepoSQL(readDB, writeDB *{{if .SQLUseGORM}}gorm{{else}}sql{{end}}.DB{{if not .SQLUseGORM}}, tx *sql.Tx{{end}}) {{clean (upper .ModuleName)}}Repository {
	return &{{clean .ModuleName}}RepoSQL{
		readDB, writeDB,{{if not .SQLUseGORM}} tx,{{end}}
	}
}

func (r *{{clean .ModuleName}}RepoSQL) FetchAll(ctx context.Context, filter *candishared.Filter) (data []shareddomain.{{clean (upper .ModuleName)}}, err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoSQL:FetchAll")
	defer func() { trace.SetError(err); trace.Finish() }()

	if filter.OrderBy == "" {
		filter.OrderBy = ` + `"modified_at"` + `
	}
	
	{{if .SQLUseGORM}}err = r.readDB.
		Order(filter.OrderBy + " " + filter.Sort).
		Limit(filter.Limit).Offset(filter.Offset).
		Find(&data).Error{{end}}
	return
}

func (r *{{clean .ModuleName}}RepoSQL) Count(ctx context.Context, filter *candishared.Filter) (count int) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoSQL:Count")
	defer trace.Finish()

	var total int64{{if .SQLUseGORM}}
	r.readDB.Model(&shareddomain.{{clean (upper .ModuleName)}}{}).Count(&total){{end}}
	count = int(total)
	return
}

func (r *{{clean .ModuleName}}RepoSQL) Find(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoSQL:Find")
	defer func() { trace.SetError(err); trace.Finish() }()

	return{{if .SQLUseGORM}} r.readDB.First(data).Error{{end}}
}

func (r *{{clean .ModuleName}}RepoSQL) Save(ctx context.Context, data *shareddomain.{{clean (upper .ModuleName)}}) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoSQL:Save")
	defer func() { trace.SetError(err); trace.Finish() }()
	tracer.Log(ctx, "data", data)

	data.ModifiedAt = time.Now()
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	return{{if .SQLUseGORM}} r.writeDB.Save(data).Error{{end}}
}

func (r *{{clean .ModuleName}}RepoSQL) Delete(ctx context.Context, id string) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "{{clean (upper .ModuleName)}}RepoSQL:Save")
	defer func() { trace.SetError(err); trace.Finish() }()

	return{{if .SQLUseGORM}} r.writeDB.Delete(&shareddomain.{{clean (upper .ModuleName)}}{ID: id}).Error{{end}}
}
`
)
