package problem

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNewMarshalsToRFC9457JSON(t *testing.T) {
	b, err := json.Marshal(New(http.StatusNotFound, "test not found"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"about:blank","title":"Not Found","status":404,"detail":"test not found"}`
	if string(b) != want {
		t.Fatalf("got %s, want %s", string(b), want)
	}
}
