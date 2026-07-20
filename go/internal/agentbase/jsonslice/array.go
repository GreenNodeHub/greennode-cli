package jsonslice

import "encoding/json"

// Array is a JSON array that marshals a nil underlying slice as [] and
// unmarshals JSON null to an empty slice.
type Array[T any] []T

// MarshalJSON implements json.Marshaler. Nil Array encodes as [].
func (a Array[T]) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(a))
}

// UnmarshalJSON implements json.Unmarshaler. JSON null decodes as an empty Array.
func (a *Array[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*a = Array[T]{}
		return nil
	}
	var s []T
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*a = Array[T](s)
	return nil
}
