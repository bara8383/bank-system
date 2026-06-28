package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzReturnsOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if got := recorder.Body.String(); got != healthzResponse {
		t.Fatalf("expected body %q, got %q", healthzResponse, got)
	}

	contentType := recorder.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
}

func TestHealthzRejectsUnsupportedMethods(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/healthz", nil)

	NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}

	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, allow)
	}
}

func TestHealthzResponseDoesNotExposeOperationalDetails(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewRouter().ServeHTTP(recorder, request)

	body := strings.ToLower(recorder.Body.String())
	for _, forbidden := range []string{
		"password",
		"secret",
		"token",
		"postgres",
		"database",
		"localhost",
		"stack",
		"/workspace",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("healthz response exposed forbidden detail %q in body %q", forbidden, body)
		}
	}
}
