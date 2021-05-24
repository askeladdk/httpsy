package httpsyproblem

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

type testEmbeddedDetails struct {
	Details
	ID string `json:"id"`
}

func TestEmbed(t *testing.T) {
	detail := testEmbeddedDetails{
		Details: Wrap(nil, http.StatusBadRequest),
		ID:      "myid",
	}
	detail.Detailf("invalid input")

	if detail.Error() != http.StatusText(http.StatusBadRequest) {
		t.Fatal()
	}

	if detail.StatusCode() != http.StatusBadRequest {
		t.Fatal()
	}

	if !detail.ProblemDetailer() {
		t.Fatal()
	}

	b, _ := json.Marshal(detail)
	if string(b) != `{"detail":"invalid input","status":400,"title":"Bad Request","id":"myid"}` {
		t.Fatal(string(b))
	}
}

func TestMarshalJSON(t *testing.T) {
	b, _ := json.Marshal(Details{})
	if string(b) != "{}" {
		t.Fatal(string(b))
	}
}

func TestWrap(t *testing.T) {
	var err1 error = errors.New("permission denied")
	var err2 error = Wrap(err1, http.StatusForbidden)
	if errors.Unwrap(err2) != err1 {
		t.Fatal()
	}

	js, _ := json.Marshal(err2)
	if string(js) != `{"detail":"permission denied","status":403,"title":"Forbidden"}` {
		t.Fatal(string(js))
	}
}
