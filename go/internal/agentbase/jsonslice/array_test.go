package jsonslice

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestArray_MarshalJSON_nilIsEmptyArray(t *testing.T) {
	var a Array[string]
	b, err := json.Marshal(struct {
		X Array[string] `json:"x"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"x":[]}` {
		t.Fatalf("got %s", b)
	}
	_ = a // zero value used in struct above
}

func TestArray_MarshalJSON_withElements(t *testing.T) {
	b, err := json.Marshal(struct {
		X Array[string] `json:"x"`
	}{X: Array[string]{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	x, ok := m["x"].([]any)
	if !ok || len(x) != 2 {
		t.Fatalf("got %v", m["x"])
	}
}

func TestArray_UnmarshalJSON_null(t *testing.T) {
	var a Array[int]
	if err := json.Unmarshal([]byte("null"), &a); err != nil {
		t.Fatal(err)
	}
	if a != nil && len(a) != 0 {
		t.Fatalf("expected empty, got %#v", a)
	}
}

func TestArray_UnmarshalJSON_emptyArray(t *testing.T) {
	var a Array[int]
	if err := json.Unmarshal([]byte("[]"), &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 0 {
		t.Fatalf("got %#v", a)
	}
}

func TestArray_roundTrip(t *testing.T) {
	type row struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	in := Array[row]{{ID: 1, Name: "one"}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Array[row]
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]row(in), []row(out)) {
		t.Fatalf("in %+v out %+v", in, out)
	}
}
