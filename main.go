package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
)

var DefaultBucket = []byte("Locations")

type Location json.RawMessage

type Handler struct {
	db *bolt.DB
}

func (h *Handler) AddLocations(w http.ResponseWriter, r *http.Request) {
	var loc json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)

		key := uuid.New()
		if err := b.Put([]byte(key), loc); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "[error]", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", `text/plain; charset="utf-8"`)
	fmt.Fprintln(w, "ありがとうございました！")
}

func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	locs := make([]json.RawMessage, 0)
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DefaultBucket)

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			locs = append(locs, v)
		}

		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "[error]", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", `application/json; charset="utf-8"`)
	if err := json.NewEncoder(w).Encode(locs); err != nil {
		fmt.Fprintln(os.Stderr, "[error]", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
}

func main() {
	var (
		dbPath = flag.String("db", "db.bolt", "database location")
		port   = flag.Int("port", 8080, "port to listen on")
	)
	flag.Parse()

	db, err := bolt.Open(*dbPath, 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(DefaultBucket)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	handler := Handler{db}

	r := mux.NewRouter()

	r.HandleFunc("/locations", handler.AddLocations).Methods("POST")
	r.HandleFunc("/locations", handler.ListLocations).Methods("GET")

	h := handlers.LoggingHandler(os.Stderr, r)

	addr := fmt.Sprint(":", *port)
	fmt.Println("listening at", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
