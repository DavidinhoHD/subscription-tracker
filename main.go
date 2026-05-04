package main

import (
	"log"
	"net/http"
)

const (
	listenAddr = ":8080"
	dbPath     = "subscriptions.db"
)

func main() {
	db, err := OpenDB(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	server := NewServer(db)

	log.Printf("listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, server); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
