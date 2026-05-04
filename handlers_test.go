package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCreateListAndSummaryHandlers(t *testing.T) {
	server := NewServer(openTestDB(t))

	body := `{
		"name": "Netflix",
		"amount": 9.99,
		"currency": "EUR",
		"cycle": "monthly",
		"start_date": "2024-01-15",
		"notes": ""
	}`
	rec := performRequest(server, http.MethodPost, "/api/subscriptions", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var created Subscription
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == 0 || created.Name != "Netflix" {
		t.Fatalf("created = %#v", created)
	}

	rec = performRequest(server, http.MethodGet, "/api/subscriptions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var list []Subscription
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Netflix" {
		t.Fatalf("list = %#v", list)
	}

	rec = performRequest(server, http.MethodGet, "/api/summary", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var summary Summary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if len(summary.Totals) != 1 || !floatEqual(summary.Totals[0].Monthly, 9.99) {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestUpdateAndDeleteHandlers(t *testing.T) {
	server := NewServer(openTestDB(t))

	create := performRequest(server, http.MethodPost, "/api/subscriptions", validSubscriptionJSON("Netflix"))
	var created Subscription
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	updateBody := `{
		"name": "Adobe CC",
		"amount": 75,
		"currency": "EUR",
		"cycle": "yearly",
		"start_date": "2025-03-01",
		"notes": "Creative Cloud full plan"
	}`
	rec := performRequest(server, http.MethodPut, "/api/subscriptions/"+strconv.Itoa(created.ID), updateBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var updated Subscription
	if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if updated.Name != "Adobe CC" || updated.Cycle != CycleYearly {
		t.Fatalf("updated = %#v", updated)
	}

	rec = performRequest(server, http.MethodDelete, "/api/subscriptions/"+strconv.Itoa(created.ID), "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	rec = performRequest(server, http.MethodDelete, "/api/subscriptions/"+strconv.Itoa(created.ID), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("second DELETE status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandlerErrors(t *testing.T) {
	server := NewServer(openTestDB(t))

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "validation", method: http.MethodPost, path: "/api/subscriptions", body: `{"amount":1}`, want: http.StatusBadRequest},
		{name: "invalid id", method: http.MethodPut, path: "/api/subscriptions/nope", body: validSubscriptionJSON("Netflix"), want: http.StatusBadRequest},
		{name: "missing update", method: http.MethodPut, path: "/api/subscriptions/999", body: validSubscriptionJSON("Netflix"), want: http.StatusNotFound},
		{name: "method not allowed collection", method: http.MethodDelete, path: "/api/subscriptions", want: http.StatusMethodNotAllowed},
		{name: "method not allowed summary", method: http.MethodPost, path: "/api/summary", want: http.StatusMethodNotAllowed},
		{name: "unknown route", method: http.MethodGet, path: "/api/nope", want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performRequest(server, tt.method, tt.path, tt.body)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.want, rec.Body.String())
			}
			if rec.Code != http.StatusNoContent && !strings.Contains(rec.Body.String(), `"error"`) {
				t.Fatalf("body = %q, want JSON error", rec.Body.String())
			}
		})
	}
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func validSubscriptionJSON(name string) string {
	return `{
		"name": "` + name + `",
		"amount": 9.99,
		"currency": "EUR",
		"cycle": "monthly",
		"start_date": "2024-01-15",
		"notes": ""
	}`
}
