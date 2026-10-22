package operation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// ParseBody validates one JSON body with exact numbers.
func ParseBody(cmd *cobra.Command, contract *BodyContract) (any, error) {
	if contract == nil {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString("body")
	if raw == "" {
		if contract.Kind == OptionalObjectBody || contract.Kind == OptionalJSONBody {
			return nil, nil
		}
		return nil, errors.New("body must not be empty")
	}
	value, err := parseJSONValue(raw, "body", contract.Style)
	if err != nil {
		return nil, err
	}
	switch contract.Kind {
	case ObjectBody, OptionalObjectBody:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, errors.New("body must be a JSON object")
		}
		for _, field := range contract.RequiredFields {
			if fieldValue, present := object[field]; !present || fieldValue == nil {
				return nil, fmt.Errorf("body is missing required field %q for %s", field, contract.Name)
			}
		}
	case ArrayBody:
		items, ok := value.([]any)
		if !ok {
			return nil, errors.New("body must be a JSON array")
		}
		if contract.StrictArrayItems {
			for index, item := range items {
				object, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("body item %d must be a JSON object", index)
				}
				for _, field := range contract.RequiredItemFields {
					if fieldValue, present := object[field]; !present || fieldValue == nil {
						return nil, fmt.Errorf("body item %d is missing required field %q for %s", index, field, contract.Name)
					}
				}
			}
		}
	}
	return value, nil
}

// parseJSONValue decodes exactly one JSON value with numeric precision.
func parseJSONValue(raw, name string, style BodyStyle) (any, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, decodeError(style, name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, multiValueError(style, name)
		}
		return nil, decodeError(style, name, err)
	}
	return value, nil
}

func decodeError(style BodyStyle, name string, err error) error {
	switch style {
	case BodyStyleJSONObject:
		return fmt.Errorf("invalid %s JSON object: %w", name, err)
	case BodyStyleJSON:
		return fmt.Errorf("invalid %s JSON: %w", name, err)
	default:
		return fmt.Errorf("invalid JSON %s: %w", name, err)
	}
}

func multiValueError(style BodyStyle, name string) error {
	switch style {
	case BodyStyleJSONObject:
		return fmt.Errorf("invalid %s JSON object: multiple JSON values are not allowed", name)
	case BodyStyleJSON:
		return fmt.Errorf("invalid %s JSON: multiple JSON values are not allowed", name)
	default:
		return fmt.Errorf("invalid JSON %s: multiple JSON values are not allowed", name)
	}
}

// parseJSONObject validates a query object.
func parseJSONObject(raw, name string, style QueryObjectStyle) (map[string]any, error) {
	bodyStyle := BodyStyleJSONObject
	if style == QueryObjectJSON {
		bodyStyle = BodyStyleJSON
	}
	value, err := parseJSONValue(raw, name, bodyStyle)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a JSON object", name)
	}
	return object, nil
}
