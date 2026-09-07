package vdbclient

import (
	"reflect"
	"testing"
)

// TestPlainNumbers: every JSON number decodes to float64, and the table formatter
// prints cells with %v, which switches to exponent form once that is shorter —
// a 5861298-byte backup rendered as "5.861298e+06". Table output converts first.
func TestPlainNumbers(t *testing.T) {
	got := plainNumbers(map[string]interface{}{
		"totalBackupSize": 5861298.0,
		"volumeSize":      100.0,
		"backupUsage":     0.06,
		"name":            "keep-me",
		"enabled":         true,
		"missing":         nil,
		"nested": []interface{}{
			map[string]interface{}{"uncompressedSize": 31677251.0},
		},
	}).(map[string]interface{})

	for field, want := range map[string]interface{}{
		"totalBackupSize": "5861298",
		"volumeSize":      "100",
		"backupUsage":     "0.06",
		"name":            "keep-me",
		"enabled":         true,
		"missing":         nil,
	} {
		if got[field] != want {
			t.Errorf("%s = %#v, want %#v", field, got[field], want)
		}
	}

	nested := got["nested"].([]interface{})[0].(map[string]interface{})
	if nested["uncompressedSize"] != "31677251" {
		t.Errorf("nested number = %#v, want a plain decimal string", nested["uncompressedSize"])
	}
}

func TestUnwrap(t *testing.T) {
	listPayload := map[string]interface{}{
		"projectId": "pro-1",
		"data": []interface{}{
			map[string]interface{}{"id": "db-1", "name": "one"},
		},
		"pageObject": map[string]interface{}{"totalPages": 2.0},
	}

	tests := []struct {
		name string
		in   interface{}
		want interface{}
	}{
		{
			// The list envelope nests data inside data; unwrapping the outer
			// layer keeps both the items and pageObject.
			name: "paginated list envelope",
			in: map[string]interface{}{
				"code": 200.0, "message": "Success", "data": listPayload,
			},
			want: listPayload,
		},
		{
			// Backups, configurations and histories key their items "content",
			// not "data". Unwrap must not care which — it only strips the outer
			// envelope and hands the payload through as-is.
			name: "paginated list keyed by content",
			in: map[string]interface{}{
				"code": 200.0, "message": "Success",
				"data": map[string]interface{}{
					"content":    []interface{}{map[string]interface{}{"id": "bkp-1"}},
					"pageObject": map[string]interface{}{"totalPages": 1.0},
				},
			},
			want: map[string]interface{}{
				"content":    []interface{}{map[string]interface{}{"id": "bkp-1"}},
				"pageObject": map[string]interface{}{"totalPages": 1.0},
			},
		},
		{
			// 76 of the wrapped endpoints put a bare array directly in data.
			name: "envelope around a bare array",
			in: map[string]interface{}{
				"code": 200.0, "message": "Success",
				"data": []interface{}{map[string]interface{}{"id": "sg-1"}},
			},
			want: []interface{}{map[string]interface{}{"id": "sg-1"}},
		},
		{
			name: "detail envelope",
			in: map[string]interface{}{
				"code": 200.0, "message": "Success",
				"data": map[string]interface{}{"id": "db-1"},
			},
			want: map[string]interface{}{"id": "db-1"},
		},
		{
			// Kafka returns bare arrays with no envelope — pass through untouched.
			name: "bare array is untouched",
			in:   []interface{}{map[string]interface{}{"id": "kfk-1"}},
			want: []interface{}{map[string]interface{}{"id": "kfk-1"}},
		},
		{
			// A payload that merely has a "data" key is not an envelope; without
			// code+message we must not strip a real field off the response.
			name: "data key without code and message is not an envelope",
			in:   map[string]interface{}{"data": "keep me", "other": 1.0},
			want: map[string]interface{}{"data": "keep me", "other": 1.0},
		},
		{
			name: "envelope with null data is left alone",
			in: map[string]interface{}{
				"code": 204.0, "message": "No Content", "data": nil,
			},
			want: map[string]interface{}{
				"code": 204.0, "message": "No Content", "data": nil,
			},
		},
		{
			name: "envelope missing data is left alone",
			in:   map[string]interface{}{"code": 400.0, "message": "Bad Request"},
			want: map[string]interface{}{"code": 400.0, "message": "Bad Request"},
		},
		{
			name: "non-map input is untouched",
			in:   "plain string",
			want: "plain string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unwrap(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unwrap() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestPayloadDetectsTheNullDataMiss: vDB answers "not found" with HTTP 200 and a
// null payload on some endpoints, so the status code cannot be relied on. Unwrap
// hands back the envelope in that case, which a caller would happily read empty
// fields off — Payload is what makes the miss visible.
func TestPayloadDetectsTheNullDataMiss(t *testing.T) {
	// The real response for a backup whose creation failed.
	miss := map[string]interface{}{"code": 200.0, "message": "success", "data": nil}

	if _, ok := Payload(miss); ok {
		t.Error("Payload reported success for a null data payload")
	}
	if _, ok := PayloadObject(miss); ok {
		t.Error("PayloadObject reported success for a null data payload")
	}
	// Unwrap on its own cannot tell: it returns the envelope, whose fields read empty.
	if unwrapped, ok := Unwrap(miss).(map[string]interface{}); !ok || unwrapped["status"] != nil {
		t.Error("assumption changed: Unwrap no longer returns the envelope for null data")
	}

	// A real object comes through.
	hit := map[string]interface{}{
		"code": 200.0, "message": "ok",
		"data": map[string]interface{}{"id": "bk-1", "status": "COMPLETED"},
	}
	object, ok := PayloadObject(hit)
	if !ok || object["id"] != "bk-1" {
		t.Errorf("PayloadObject = %v, %v", object, ok)
	}

	// Non-envelope payloads (bare arrays, the kafka style) pass through.
	if payload, ok := Payload([]interface{}{"a"}); !ok || len(payload.([]interface{})) != 1 {
		t.Errorf("bare array = %v, %v", payload, ok)
	}
	// An envelope whose data is an empty object is a hit, not a miss: the resource
	// exists and simply has no fields set.
	empty := map[string]interface{}{"code": 200.0, "message": "ok", "data": map[string]interface{}{}}
	if _, ok := PayloadObject(empty); !ok {
		t.Error("an empty object payload should count as found")
	}
}

// TestExtractIDValuesTakesNumbers: several vDB ids are JSON numbers — a flavor's id and a
// backup package's packageId — and cli.ExtractIDs skips those, which left --package-id
// completion silently empty until it was tried live.
func TestExtractIDValuesTakesNumbers(t *testing.T) {
	payload := []interface{}{
		map[string]interface{}{"id": float64(179)},
		map[string]interface{}{"id": "cfg-1"},
		map[string]interface{}{"id": float64(179)},  // duplicate: one flavor name, two rows
		map[string]interface{}{"id": float64(1e6)},  // must not come out as 1e+06
		map[string]interface{}{"id": float64(20.5)}, // fractional values keep their point
		map[string]interface{}{"id": nil},           // skipped
		map[string]interface{}{"other": "x"},        // field absent
		map[string]interface{}{"id": ""},            // empty string is not a suggestion
		"not a row",
	}

	got := ExtractIDValues(payload, "id")
	want := []string{"179", "cfg-1", "1000000", "20.5"}
	if len(got) != len(want) {
		t.Fatalf("ExtractIDValues = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ExtractIDValues[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if ExtractIDValues(map[string]interface{}{"id": float64(1)}, "id") != nil {
		t.Error("a non-list payload must yield nothing")
	}
}
