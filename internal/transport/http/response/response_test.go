package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestJSON проверяет запись произвольного JSON-ответа.
func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	payload := map[string]string{
		"status": "ok",
	}

	JSON(rec, http.StatusCreated, payload)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusCreated)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: got %q, want %q", got, "application/json; charset=utf-8")
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp["status"] != "ok" {
		t.Fatalf("unexpected body field: got %q, want %q", resp["status"], "ok")
	}
}

// TestError проверяет запись JSON-ответа с ошибкой.
func TestError(t *testing.T) {
	rec := httptest.NewRecorder()

	Error(rec, http.StatusBadRequest, "invalid request")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected Content-Type: got %q, want %q", got, "application/json; charset=utf-8")
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}

	if resp.Error != "invalid request" {
		t.Fatalf("unexpected error message: got %q, want %q", resp.Error, "invalid request")
	}
}
