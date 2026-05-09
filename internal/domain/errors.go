package domain

import "fmt"

// ParseError is returned when a CSV row cannot be parsed.
type ParseError struct {
	File       string
	LineNumber int
	Field      string
	Message    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %s line %d (field: %s): %s",
		e.File, e.LineNumber, e.Field, e.Message)
}

// ValidationError is returned when a parsed value violates business rules.
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s (value: %q): %s",
		e.Field, e.Value, e.Message)
}

// ConfigError is returned for invalid CLI configuration.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error: %s", e.Message)
}
