package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestSubscriptionRepository(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	initial, err := ListSubscriptions(ctx, db)
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(initial) != 0 {
		t.Fatalf("len(initial) = %d, want 0", len(initial))
	}

	created, err := InsertSubscription(ctx, db, Subscription{
		Name:      "Netflix",
		Amount:    9.99,
		Currency:  "EUR",
		Cycle:     CycleMonthly,
		StartDate: "2024-01-15",
		Notes:     "",
	})
	if err != nil {
		t.Fatalf("InsertSubscription() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created.ID = 0, want generated id")
	}

	got, err := GetSubscription(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if got.Name != "Netflix" || got.Currency != "EUR" {
		t.Fatalf("GetSubscription() = %#v", got)
	}

	updated, err := UpdateSubscription(ctx, db, created.ID, Subscription{
		Name:      "Adobe CC",
		Amount:    75,
		Currency:  "EUR",
		Cycle:     CycleYearly,
		StartDate: "2025-03-01",
		Notes:     "Creative Cloud full plan",
	})
	if err != nil {
		t.Fatalf("UpdateSubscription() error = %v", err)
	}
	if updated.Name != "Adobe CC" || updated.Cycle != CycleYearly || updated.Notes == "" {
		t.Fatalf("UpdateSubscription() = %#v", updated)
	}

	list, err := ListSubscriptions(ctx, db)
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	if _, err := UpdateSubscription(ctx, db, 999, updated); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateSubscription() missing error = %v, want ErrNotFound", err)
	}
	if err := DeleteSubscription(ctx, db, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteSubscription() missing error = %v, want ErrNotFound", err)
	}

	if err := DeleteSubscription(ctx, db, created.ID); err != nil {
		t.Fatalf("DeleteSubscription() error = %v", err)
	}
	if _, err := GetSubscription(ctx, db, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetSubscription() after delete error = %v, want ErrNotFound", err)
	}
}

func TestSubscriptionRepositoryRejectsInvalidCycle(t *testing.T) {
	db := openTestDB(t)

	_, err := InsertSubscription(context.Background(), db, Subscription{
		Name:      "Bad",
		Amount:    1,
		Currency:  "EUR",
		Cycle:     "weekly",
		StartDate: "2024-01-01",
	})
	if err == nil {
		t.Fatal("expected error for invalid cycle")
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenDB(filepath.Join(t.TempDir(), "subscriptions.db"))
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return db
}
