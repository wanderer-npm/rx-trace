package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type alertIn struct {
	DrugID   int64   `json:"drug_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusM  int     `json:"radius_m"`
	Channel  string  `json:"channel"`
	Target   string  `json:"target"`
}

type alertRow struct {
	ID          int64      `json:"id"`
	DrugID      int64      `json:"drug_id"`
	Drug        string     `json:"drug"`
	Lat         float64    `json:"lat"`
	Lng         float64    `json:"lng"`
	RadiusM     int        `json:"radius_m"`
	Channel     string     `json:"channel"`
	LastFiredAt *time.Time `json:"last_fired_at,omitempty"`
	Active      bool       `json:"active"`
}

func (h *Handlers) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var body alertIn
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	if body.DrugID == 0 || body.Target == "" {
		writeErr(w, 400, "drug_id and target required")
		return
	}
	if body.RadiusM <= 0 || body.RadiusM > 25000 {
		body.RadiusM = 5000
	}
	switch body.Channel {
	case "email":
		if !strings.Contains(body.Target, "@") {
			writeErr(w, 400, "invalid email")
			return
		}
	case "webhook":
		if !(strings.HasPrefix(body.Target, "http://") || strings.HasPrefix(body.Target, "https://")) {
			writeErr(w, 400, "invalid webhook url")
			return
		}
	case "push":
	default:
		writeErr(w, 400, "channel must be email, push or webhook")
		return
	}

	var one int
	if err := h.DB.QueryRow(r.Context(), "select 1 from drugs where id = $1", body.DrugID).Scan(&one); err != nil {
		writeErr(w, 404, "drug not found")
		return
	}
	var id int64
	err := h.DB.QueryRow(r.Context(), `
        insert into alerts (drug_id, lat, lng, radius_m, channel, target)
        values ($1,$2,$3,$4,$5,$6) returning id`,
		body.DrugID, body.Lat, body.Lng, body.RadiusM, body.Channel, body.Target,
	).Scan(&id)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]int64{"id": id})
}

func (h *Handlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if len(target) < 3 {
		writeErr(w, 400, "target required")
		return
	}
	rows, err := h.DB.Query(r.Context(), `
        select a.id, a.drug_id, d.name, a.lat, a.lng, a.radius_m,
               a.channel, a.last_fired_at, a.active
        from alerts a
        join drugs d on d.id = a.drug_id
        where a.target = $1
        order by a.created_at desc`, target)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := []alertRow{}
	for rows.Next() {
		var a alertRow
		var last sql.NullTime
		if err := rows.Scan(&a.ID, &a.DrugID, &a.Drug, &a.Lat, &a.Lng, &a.RadiusM, &a.Channel, &last, &a.Active); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if last.Valid {
			t := last.Time
			a.LastFiredAt = &t
		}
		out = append(out, a)
	}
	writeJSON(w, 200, out)
}

func (h *Handlers) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	target := r.URL.Query().Get("target")
	if target == "" {
		writeErr(w, 400, "target required")
		return
	}
	tag, err := h.DB.Exec(r.Context(), "delete from alerts where id = $1 and target = $2", id, target)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, map[string]bool{"deleted": true})
}
