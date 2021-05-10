package validator

import "github.com/golangid/candi/config/env"

// Validator instance
type Validator struct {
	*JSONSchemaValidator
	*StructValidator
}

// NewValidator constructor, using jsonschema & struct validator (github.com/go-playground/validator),
// jsonschema source file load from JSON_SCHEMA_DIR environment
func NewValidator() *Validator {
	return &Validator{
		JSONSchemaValidator: NewJSONSchemaValidator(env.BaseEnv().JSONSchemaDir),
		StructValidator:     NewStructValidator(),
	}
}
