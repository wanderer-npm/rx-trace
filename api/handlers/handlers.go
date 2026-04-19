package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handlers struct {
	DB *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Handlers {
	return &Handlers{DB: pool}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type Drug struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Generic  string   `json:"generic,omitempty"`
	Form     string   `json:"form,omitempty"`
	Strength string   `json:"strength,omitempty"`
	Tags     []string `json:"tags"`
}

type Pharmacy struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address,omitempty"`
	City      string  `json:"city,omitempty"`
	Phone     string  `json:"phone,omitempty"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	DistanceM float64 `json:"distance_m,omitempty"`
	Verified  bool    `json:"verified"`
}

func score(q, name, generic string) int {
	q = strings.ToLower(q)
	name = strings.ToLower(name)
	generic = strings.ToLower(generic)
	switch {
	case name == q:
		return 1000
	case strings.HasPrefix(name, q):
		return 800 - len(name)
	case strings.HasPrefix(generic, q):
		return 600 - len(generic)
	case strings.Contains(name, q):
		return 400 - len(name)
	case strings.Contains(generic, q):
		return 200 - len(generic)
	}
	return 0
}
