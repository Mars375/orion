package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ContractValidator validates messages against JSON Schema contracts.
//
// Implements contract-first design pattern from ORION architecture:
//   - All messages MUST validate against schemas before publish
//   - Invalid messages are rejected immediately (fail-fast)
//   - Validation errors include full details for debugging
type ContractValidator struct {
	schemas map[string]*jsonschema.Schema
	logger  *log.Logger
}

// NewContractValidator creates a new validator from a contracts directory.
//
// Loads all *.schema.json files from the specified directory and compiles
// them using JSON Schema Draft 2020-12. Schema keys are derived from filenames
// (e.g., "event.schema.json" -> "event").
//
// Parameters:
//   - contractsDir: Path to directory containing *.schema.json files
//
// Returns:
//   - Initialized ContractValidator
//   - Error if directory doesn't exist or schemas fail to compile
//
// Example:
//
//	validator, err := NewContractValidator("../contracts")
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewContractValidator(contractsDir string) (*ContractValidator, error) {
	v := &ContractValidator{
		schemas: make(map[string]*jsonschema.Schema),
		logger:  log.Default(),
	}

	// Find all schema files
	pattern := filepath.Join(contractsDir, "*.schema.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find schema files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no schema files found in %s", contractsDir)
	}

	// Load and compile each schema
	for _, file := range files {
		schema, err := v.loadSchema(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}

		// Extract contract type from filename (remove .schema.json suffix)
		basename := filepath.Base(file)
		contractType := strings.TrimSuffix(basename, ".schema.json")
		v.schemas[contractType] = schema

		v.logger.Printf("Loaded schema: %s", contractType)
	}

	return v, nil
}

// Validate validates a message against the specified contract type.
//
// Performs JSON Schema validation according to Draft 2020-12. Messages that
// fail validation return detailed error information for debugging.
//
// Parameters:
//   - message: Message to validate (will be marshaled to JSON internally)
//   - contractType: Contract type (e.g., "event", "incident", "decision")
//
// Returns:
//   - nil if message is valid
//   - error with validation details if message is invalid or contract type unknown
//
// Example:
//
//	msg := map[string]interface{}{
//	    "version": "1.0",
//	    "event_id": "550e8400-e29b-41d4-a716-446655440000",
//	    // ... other fields
//	}
//	if err := validator.Validate(msg, "event"); err != nil {
//	    log.Fatalf("Invalid event: %v", err)
//	}
func (v *ContractValidator) Validate(message map[string]interface{}, contractType string) error {
	// Look up schema
	schema, ok := v.schemas[contractType]
	if !ok {
		return fmt.Errorf("unknown contract type: %s", contractType)
	}

	// Validate message against schema
	if err := schema.Validate(message); err != nil {
		return fmt.Errorf("validation failed for %s: %w", contractType, err)
	}

	return nil
}

// loadSchema loads and compiles a JSON Schema file.
//
// Reads the schema file and compiles it using jsonschema library with
// Draft 2020-12 support.
//
// Parameters:
//   - path: Path to .schema.json file
//
// Returns:
//   - Compiled schema
//   - Error if file cannot be read or schema is invalid
func (v *ContractValidator) loadSchema(path string) (*jsonschema.Schema, error) {
	// Read schema file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse JSON
	var schemaDoc interface{}
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Compile schema
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return schema, nil
}
