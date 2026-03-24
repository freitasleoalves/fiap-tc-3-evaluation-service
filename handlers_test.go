package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Evaluation(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	app.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperado status 200, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("esperado status 'ok', got '%s'", body["status"])
	}
}

func TestEvaluationHandler_MissingParams(t *testing.T) {
	app := &App{}

	tests := []struct {
		name  string
		query string
	}{
		{"sem params", ""},
		{"sem user_id", "?flag_name=test"},
		{"sem flag_name", "?user_id=user1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/evaluate"+tt.query, nil)
			w := httptest.NewRecorder()

			app.evaluationHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("esperado status 400, got %d", w.Code)
			}
		})
	}
}
