package config

import (
	"encoding/json"
)

// UnmarshalJSON unmarshals JSON data into a value.
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
