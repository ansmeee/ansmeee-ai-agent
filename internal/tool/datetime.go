package tool

import (
	"context"
	"fmt"
	"time"
)

// DateTime provides current date/time information.
type DateTime struct{}

// Name returns the tool name.
func (d *DateTime) Name() string { return "datetime" }

// Description returns the tool description.
func (d *DateTime) Description() string {
	return "Get the current date and time. Pass \"now\" for current time, \"date\" for current date, \"weekday\" for day of week, or \"timestamp\" for Unix timestamp."
}

// Parameters returns the JSON Schema for the tool's input.
func (d *DateTime) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"format": map[string]any{
				"type":        "string",
				"description": "Output format: \"now\" (RFC3339), \"date\" (YYYY-MM-DD), \"weekday\", or \"timestamp\" (Unix)",
				"enum":        []string{"now", "date", "weekday", "timestamp"},
			},
		},
		"required": []string{"format"},
	}
}

// Call returns date/time info based on the input.
func (d *DateTime) Call(ctx context.Context, input string) (string, error) {
	now := time.Now()
	switch input {
	case "now", "":
		return now.Format(time.RFC3339), nil
	case "date":
		return now.Format("2006-01-02"), nil
	case "weekday":
		return now.Weekday().String(), nil
	case "timestamp":
		return fmt.Sprintf("%d", now.Unix()), nil
	default:
		return now.Format(time.RFC3339), nil
	}
}
