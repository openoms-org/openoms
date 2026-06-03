package model

import (
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Provider field-schema group keys (design spec §118-137).
const (
	FieldGroupSecretCredentials = "secret_credentials"
	FieldGroupSettings          = "settings"
	FieldGroupEnvironment       = "environment"
	FieldGroupSync              = "sync"
	FieldGroupFeatureToggles    = "feature_toggles"
	FieldGroupProviderOptions   = "provider_options"
)

// Provider field types.
const (
	FieldTypeString   = "string"
	FieldTypePassword = "password"
	FieldTypeNumber   = "number"
	FieldTypeBoolean  = "boolean"
	FieldTypeEnum     = "enum"
	FieldTypeURL      = "url"
	FieldTypeTextarea = "textarea"
)

// Environment scopes for a field.
const (
	FieldEnvAll        = "all"
	FieldEnvProduction = "production"
	FieldEnvSandbox    = "sandbox"
)

var validProviderFieldGroupKeys = []string{
	FieldGroupSecretCredentials, FieldGroupSettings, FieldGroupEnvironment,
	FieldGroupSync, FieldGroupFeatureToggles, FieldGroupProviderOptions,
}

var validProviderFieldTypes = []string{
	FieldTypeString, FieldTypePassword, FieldTypeNumber, FieldTypeBoolean,
	FieldTypeEnum, FieldTypeURL, FieldTypeTextarea,
}

var validProviderFieldEnvScopes = []string{"", FieldEnvAll, FieldEnvProduction, FieldEnvSandbox}

// ProviderFieldValidation declares per-field validation rules.
type ProviderFieldValidation struct {
	Enum      []string `json:"enum,omitempty"`
	Regex     string   `json:"regex,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
}

// ProviderField is one configuration field.
type ProviderField struct {
	Key                      string                  `json:"key"`
	Label                    string                  `json:"label"`
	Type                     string                  `json:"type"`
	Required                 bool                    `json:"required"`
	Secret                   bool                    `json:"secret"`
	EnvironmentScope         string                  `json:"environment_scope,omitempty"`
	HelpText                 string                  `json:"help_text,omitempty"`
	Validation               ProviderFieldValidation `json:"validation"`
	CapabilityEnabled        string                  `json:"capability_enabled,omitempty"`
	TestConnectionDependency bool                    `json:"test_connection_dependency,omitempty"`
}

// ProviderFieldGroup groups related fields (e.g. secret_credentials).
type ProviderFieldGroup struct {
	Key    string          `json:"key"`
	Label  string          `json:"label"`
	Fields []ProviderField `json:"fields"`
}

// ProviderFieldSchema is the versioned credential/settings schema for a provider
// version. It is the single source of truth for generated forms.
type ProviderFieldSchema struct {
	ID                uuid.UUID            `json:"id"`
	ProviderVersionID uuid.UUID            `json:"provider_version_id"`
	Groups            []ProviderFieldGroup `json:"groups"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

// ValidateFieldSchema checks the structural validity of a field schema: known
// group keys, known field types/env-scopes, field keys unique across the whole
// schema, enum fields carry values, regexes compile, and numeric/length bounds
// are ordered.
func ValidateFieldSchema(groups []ProviderFieldGroup) error {
	seenKeys := map[string]bool{}
	for _, g := range groups {
		if !slices.Contains(validProviderFieldGroupKeys, g.Key) {
			return fmt.Errorf("unknown field group %q", g.Key)
		}
		for fi, f := range g.Fields {
			if f.Key == "" {
				return fmt.Errorf("group %q field #%d has an empty key", g.Key, fi)
			}
			if seenKeys[f.Key] {
				return fmt.Errorf("duplicate field key %q", f.Key)
			}
			seenKeys[f.Key] = true

			if !slices.Contains(validProviderFieldTypes, f.Type) {
				return fmt.Errorf("field %q has unknown type %q", f.Key, f.Type)
			}
			if !slices.Contains(validProviderFieldEnvScopes, f.EnvironmentScope) {
				return fmt.Errorf("field %q has unknown environment_scope %q", f.Key, f.EnvironmentScope)
			}
			if f.Type == FieldTypeEnum && len(f.Validation.Enum) == 0 {
				return fmt.Errorf("enum field %q must declare at least one value", f.Key)
			}
			if f.Validation.Regex != "" {
				if _, err := regexp.Compile(f.Validation.Regex); err != nil {
					return fmt.Errorf("field %q has an invalid regex: %w", f.Key, err)
				}
			}
			if f.Validation.Min != nil && f.Validation.Max != nil && *f.Validation.Min > *f.Validation.Max {
				return fmt.Errorf("field %q has min > max", f.Key)
			}
			if f.Validation.MinLength != nil && f.Validation.MaxLength != nil && *f.Validation.MinLength > *f.Validation.MaxLength {
				return fmt.Errorf("field %q has min_length > max_length", f.Key)
			}
		}
	}
	return nil
}
