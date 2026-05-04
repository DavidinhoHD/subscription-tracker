package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const subscriptionsPath = "/api/subscriptions"

type Server struct {
	DB *sql.DB
}

func NewServer(db *sql.DB) *Server {
	return &Server{DB: db}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == subscriptionsPath:
		s.handleSubscriptions(w, r)
	case strings.HasPrefix(r.URL.Path, subscriptionsPath+"/"):
		s.handleSubscriptionByID(w, r)
	case r.URL.Path == "/api/summary":
		s.handleSummary(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		subscriptions, err := ListSubscriptions(r.Context(), s.DB)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, subscriptions)
	case http.MethodPost:
		sub, ok := decodeAndValidateSubscription(w, r)
		if !ok {
			return
		}

		created, err := InsertSubscription(r.Context(), s.DB, sub)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseSubscriptionID(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		sub, ok := decodeAndValidateSubscription(w, r)
		if !ok {
			return
		}

		updated, err := UpdateSubscription(r.Context(), s.DB, id, sub)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		err := DeleteSubscription(r.Context(), s.DB, id)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "PUT, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	subscriptions, err := ListSubscriptions(r.Context(), s.DB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	summary, err := BuildSummary(subscriptions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func decodeAndValidateSubscription(w http.ResponseWriter, r *http.Request) (Subscription, bool) {
	var sub Subscription
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&sub); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return Subscription{}, false
	}

	if err := validateSubscription(&sub); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return Subscription{}, false
	}

	return sub, true
}

func validateSubscription(sub *Subscription) error {
	sub.Name = strings.TrimSpace(sub.Name)
	sub.Currency = strings.TrimSpace(sub.Currency)

	if sub.Name == "" {
		return fmt.Errorf("name is required")
	}
	if sub.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if sub.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if !IsValidCycle(sub.Cycle) {
		return fmt.Errorf("cycle must be one of monthly, yearly, 2yearly")
	}
	if sub.StartDate == "" {
		return fmt.Errorf("start_date is required")
	}
	if _, err := time.Parse("2006-01-02", sub.StartDate); err != nil {
		return fmt.Errorf("start_date must be YYYY-MM-DD")
	}

	return nil
}

func parseSubscriptionID(path string) (int, bool) {
	rawID := strings.TrimPrefix(path, subscriptionsPath+"/")
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, false
	}

	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
