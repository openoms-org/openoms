package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrF(f float64) *float64 { return &f }
func ptrI(i int) *int         { return &i }

func TestValidateFieldSchema_Valid(t *testing.T) {
	groups := []ProviderFieldGroup{
		{Key: FieldGroupSecretCredentials, Label: "Credentials", Fields: []ProviderField{
			{Key: "api_key", Label: "API Key", Type: FieldTypePassword, Required: true, Secret: true, TestConnectionDependency: true},
		}},
		{Key: FieldGroupSettings, Label: "Settings", Fields: []ProviderField{
			{Key: "region", Label: "Region", Type: FieldTypeEnum, EnvironmentScope: FieldEnvAll,
				Validation: ProviderFieldValidation{Enum: []string{"PL", "DE"}}},
			{Key: "timeout", Label: "Timeout", Type: FieldTypeNumber,
				Validation: ProviderFieldValidation{Min: ptrF(1), Max: ptrF(60)}},
			{Key: "endpoint", Label: "Endpoint", Type: FieldTypeURL,
				Validation: ProviderFieldValidation{Regex: `^https://`, MinLength: ptrI(8), MaxLength: ptrI(255)}},
		}},
	}
	require.NoError(t, ValidateFieldSchema(groups))
}

func TestValidateFieldSchema_Invalid(t *testing.T) {
	cases := map[string][]ProviderFieldGroup{
		"unknown group": {{Key: "bogus", Fields: nil}},
		"duplicate key": {
			{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "x", Type: FieldTypeString}}},
			{Key: FieldGroupSync, Fields: []ProviderField{{Key: "x", Type: FieldTypeString}}},
		},
		"empty key":      {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "", Type: FieldTypeString}}}},
		"unknown type":   {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: "banana"}}}},
		"unknown scope":  {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: FieldTypeString, EnvironmentScope: "moon"}}}},
		"enum no values": {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: FieldTypeEnum}}}},
		"bad regex":      {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: FieldTypeString, Validation: ProviderFieldValidation{Regex: "("}}}}},
		"min>max":        {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: FieldTypeNumber, Validation: ProviderFieldValidation{Min: ptrF(10), Max: ptrF(1)}}}}},
		"minlen>maxlen":  {{Key: FieldGroupSettings, Fields: []ProviderField{{Key: "y", Type: FieldTypeString, Validation: ProviderFieldValidation{MinLength: ptrI(10), MaxLength: ptrI(1)}}}}},
	}
	for name, groups := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, ValidateFieldSchema(groups))
		})
	}
}
