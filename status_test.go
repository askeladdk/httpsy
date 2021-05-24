package httpsy

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

type testStatusTimeout struct{}

func (t testStatusTimeout) Error() string { return "" }
func (t testStatusTimeout) Timeout() bool { return true }

type testStatusTemporary struct{}

func (t testStatusTemporary) Error() string   { return "" }
func (t testStatusTemporary) Temporary() bool { return true }

func TestStatusCode(t *testing.T) {
	testCases := []struct {
		err    error
		status int
	}{
		{nil, http.StatusOK},
		{StatusBadGateway, http.StatusBadGateway},
		{errors.New(""), http.StatusInternalServerError},
		{testStatusTimeout{}, http.StatusGatewayTimeout},
		{testStatusTemporary{}, http.StatusServiceUnavailable},
	}

	for _, testCase := range testCases {
		if StatusCode(testCase.err) != testCase.status {
			t.Fatal(testCase.err)
		}
	}
}

func TestStatusMarshal(t *testing.T) {
	b, _ := json.Marshal(StatusForbidden)
	if string(b) != `{"status":403,"title":"Forbidden"}` {
		t.Fatal(string(b))
	}
}
