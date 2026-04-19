package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
)

type Shortage struct {
	ID          int64  `json:"id"`
	Drug        string `json:"drug"`
	Source      string `json:"source"`
	Severity    string `json:"severity,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	ExpectedEnd string `json:"expected_end,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (h *Handlers) ActiveShortages(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := h.DB.Query(r.Context(), `
        select s.id,
               coalesce(d.name, '') as drug,
               s.source,
               coalesce(s.severity,''),
               to_char(s.started_at, 'YYYY-MM-DD'),
               to_char(s.expected_end, 'YYYY-MM-DD'),
               coalesce(s.url, '')
        from shortages s
        left join drugs d on d.id = s.drug_id
        where s.expected_end is null or s.expected_end > current_date
        order by s.ingested_at desc
        limit $1`, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := []Shortage{}
	for rows.Next() {
		var s Shortage
		var started, until sql.NullString
		if err := rows.Scan(&s.ID, &s.Drug, &s.Source, &s.Severity, &started, &until, &s.URL); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if started.Valid {
			s.StartedAt = started.String
		}
		if until.Valid {
			s.ExpectedEnd = until.String
		}
		out = append(out, s)
	}
	writeJSON(w, 200, out)
}
