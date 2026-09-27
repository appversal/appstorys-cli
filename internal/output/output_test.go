package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteJSONNilSliceIsEmptyArray(t *testing.T) {
	var none []string
	var buf bytes.Buffer
	if err := WriteJSON(&buf, none); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"data": []`) {
		t.Errorf("nil slice should encode as [], got:\n%s", out)
	}
	if !strings.Contains(out, `"schemaVersion": 1`) {
		t.Errorf("missing schemaVersion:\n%s", out)
	}
}

func TestWriteJSONNonSlicePayloadUnchanged(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, map[string]int{"a": 1}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if !strings.Contains(buf.String(), `"a": 1`) {
		t.Errorf("payload not written:\n%s", buf.String())
	}
}
