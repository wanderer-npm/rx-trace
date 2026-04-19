package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) SearchDrugs(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeErr(w, 400, "q must be at least 2 chars")
		return
	}
	pat := "%" + q + "%"
	rows, err := h.DB.Query(r.Context(), `
        select id, name, coalesce(generic,''), coalesce(form,''), coalesce(strength,''), coalesce(tags, '{}')
        from drugs
        where name ilike $1 or generic ilike $1
        limit 200`, pat)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	type scored struct {
		d Drug
		s int
	}
	var out []scored
	for rows.Next() {
		var d Drug
		if err := rows.Scan(&d.ID, &d.Name, &d.Generic, &d.Form, &d.Strength, &d.Tags); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		out = append(out, scored{d: d, s: score(q, d.Name, d.Generic)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].s != out[j].s {
			return out[i].s > out[j].s
		}
		return len(out[i].d.Name) < len(out[j].d.Name)
	})
	limit := 25
	if len(out) < limit {
		limit = len(out)
	}
	drugs := make([]Drug, limit)
	for i := 0; i < limit; i++ {
		drugs[i] = out[i].d
	}
	writeJSON(w, 200, drugs)
}

func (h *Handlers) GetDrug(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	var d Drug
	err = h.DB.QueryRow(r.Context(), `
        select id, name, coalesce(generic,''), coalesce(form,''), coalesce(strength,''), coalesce(tags, '{}')
        from drugs where id = $1`, id).Scan(&d.ID, &d.Name, &d.Generic, &d.Form, &d.Strength, &d.Tags)
	if err != nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, d)
}
