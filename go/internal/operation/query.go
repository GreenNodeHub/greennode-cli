package operation

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/greennodehub/greennode-cli/internal/validator"
	"github.com/spf13/cobra"
)

// BuildQuery applies defaults and validation; absent optional parameters are omitted.
func BuildQuery(cmd *cobra.Command, params []QueryParam) (map[string]string, error) {
	if len(params) == 0 {
		return nil, nil
	}
	query := make(map[string]string)
	for _, parameter := range params {
		value, _ := cmd.Flags().GetString(parameter.Flag)
		if value == "" && parameter.Default != "" {
			value = parameter.Default
		}
		if value == "" {
			if parameter.Required {
				return nil, fmt.Errorf("%s must not be empty", parameter.Flag)
			}
			continue
		}
		encoded, err := encodeQueryValue(parameter, value)
		if err != nil {
			return nil, err
		}
		query[parameter.WireName] = encoded
	}
	if len(query) == 0 {
		return nil, nil
	}
	return query, nil
}

func encodeQueryValue(parameter QueryParam, value string) (string, error) {
	switch parameter.Kind {
	case QueryInteger:
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid %s: must be a 32-bit integer", parameter.Flag)
		}
		if parameter.Minimum > 0 && parsed < parameter.Minimum {
			return "", fmt.Errorf("invalid %s: must be at least %d", parameter.Flag, parameter.Minimum)
		}
		return value, nil
	case QueryBoolean:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", fmt.Errorf("invalid %s: must be true or false", parameter.Flag)
		}
		return strconv.FormatBool(parsed), nil
	case QueryID:
		if err := validator.ValidateID(value, parameter.Flag); err != nil {
			return "", err
		}
		return value, nil
	case QueryObject:
		object, err := parseJSONObject(value, parameter.Flag, parameter.ObjectStyle)
		if err != nil {
			return "", err
		}
		encoded, err := json.Marshal(object)
		if err != nil {
			if parameter.EncodeError != nil {
				return "", parameter.EncodeError(parameter.Flag, err)
			}
			return "", fmt.Errorf("invalid %s JSON object: %w", parameter.Flag, err)
		}
		return string(encoded), nil
	default:
		return value, nil
	}
}
