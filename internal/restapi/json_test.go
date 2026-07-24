package restapi

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteError_HidesInternalDetailsFor5xx(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, 500, errors.New("raw internal db failure: connection refused"))

	body := rec.Body.String()
	if strings.Contains(body, "connection refused") {
		t.Errorf("response body leaked the raw internal error: %s", body)
	}
	if !strings.Contains(body, "internal error") {
		t.Errorf("response body = %s, want a generic \"internal error\" message", body)
	}
}

func TestWriteError_KeepsClientErrorDetailsFor4xx(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, 400, errors.New("title must not be empty"))

	body := rec.Body.String()
	if !strings.Contains(body, "title must not be empty") {
		t.Errorf("response body = %s, want the original 4xx error message preserved", body)
	}
}
