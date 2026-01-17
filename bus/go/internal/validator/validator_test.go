package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const contractsDir = "../../../contracts"

func TestNewContractValidator_LoadsAllSchemas(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)
	require.NotNil(t, validator)

	// Verify schemas were loaded (8 schemas in bus/contracts/)
	assert.GreaterOrEqual(t, len(validator.schemas), 3, "Should load at least event, incident, decision")
	assert.Contains(t, validator.schemas, "event")
	assert.Contains(t, validator.schemas, "incident")
	assert.Contains(t, validator.schemas, "decision")
}

func TestNewContractValidator_InvalidDirectory_ReturnsError(t *testing.T) {
	_, err := NewContractValidator("/nonexistent/directory")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no schema files found")
}

func TestValidate_ValidEvent_NoError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Valid event message matching event.schema.json
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data": map[string]interface{}{
			"service_name": "test-service",
		},
	}

	err = validator.Validate(message, "event")
	assert.NoError(t, err)
}

func TestValidate_MissingRequiredField_ReturnsError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Missing required field "event_id"
	message := map[string]interface{}{
		"version":    "1.0",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "info",
		"data": map[string]interface{}{
			"service_name": "test-service",
		},
	}

	err = validator.Validate(message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_InvalidSeverity_ReturnsError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Invalid severity (not in enum)
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "service_up",
		"severity":   "invalid_severity",
		"data": map[string]interface{}{
			"service_name": "test-service",
		},
	}

	err = validator.Validate(message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_UnknownContractType_ReturnsError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	message := map[string]interface{}{
		"some": "data",
	}

	err = validator.Validate(message, "unknown_type")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown contract type")
}

func TestValidate_AdditionalProperties_ReturnsError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Extra field "extra_field" that's not in schema (additionalProperties: false)
	message := map[string]interface{}{
		"version":     "1.0",
		"event_id":    "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":   "2026-01-17T12:00:00Z",
		"source":      "orion-test",
		"event_type":  "service_up",
		"severity":    "info",
		"data":        map[string]interface{}{},
		"extra_field": "not allowed",
	}

	err = validator.Validate(message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_InvalidEventType_ReturnsError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// event_type not in enum
	message := map[string]interface{}{
		"version":    "1.0",
		"event_id":   "550e8400-e29b-41d4-a716-446655440000",
		"timestamp":  "2026-01-17T12:00:00Z",
		"source":     "orion-test",
		"event_type": "invalid_event_type",
		"severity":   "info",
		"data":       map[string]interface{}{},
	}

	err = validator.Validate(message, "event")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidate_ValidIncident_NoError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Valid incident message matching incident.schema.json
	message := map[string]interface{}{
		"version":      "1.0",
		"incident_id":  "550e8400-e29b-41d4-a716-446655440001",
		"timestamp":    "2026-01-17T12:00:00Z",
		"source":       "orion-guardian",
		"incident_type": "service_outage",
		"severity":     "high",
		"event_ids": []interface{}{
			"550e8400-e29b-41d4-a716-446655440002",
		},
		"correlation_window": map[string]interface{}{
			"start": "2026-01-17T11:55:00Z",
			"end":   "2026-01-17T12:00:00Z",
		},
		"state": "open",
	}

	err = validator.Validate(message, "incident")
	assert.NoError(t, err)
}

func TestValidate_ValidDecision_NoError(t *testing.T) {
	validator, err := NewContractValidator(contractsDir)
	require.NoError(t, err)

	// Valid decision message matching decision.schema.json
	message := map[string]interface{}{
		"version":     "1.0",
		"decision_id": "550e8400-e29b-41d4-a716-446655440003",
		"timestamp":   "2026-01-17T12:00:00Z",
		"source":      "orion-brain",
		"incident_id": "550e8400-e29b-41d4-a716-446655440001",
		"decision_type": "EXECUTE_SAFE_ACTION",
		"safety_classification": "SAFE",
		"requires_approval": false,
		"proposed_action": map[string]interface{}{
			"action_type": "restart_service",
			"parameters": map[string]interface{}{
				"service_name": "test-service",
			},
		},
		"reasoning": "Service is down and requires restart",
	}

	err = validator.Validate(message, "decision")
	assert.NoError(t, err)
}
