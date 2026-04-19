package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/wanderer-npm/rx-trace/api/ratelimit"
)

type reportIn struct {
	DrugID     int64  `json:"drug_id"`
	PharmacyID int64  `json:"pharmacy_id"`
	Status     string `json:"status"`
	Note       string `json:"note"`
	PricePaise int    `json:"price_paise"`
}

type reportOut struct {
	ID           int64     `json:"id"`
	DrugID       int64     `json:"drug_id"`
	PharmacyID   int64     `json:"pharmacy_id"`
	PharmacyName string    `json:"pharmacy_name"`
	Status       string    `json:"status"`
	Note         string    `json:"note,omitempty"`
	ReportedAt   time.Time `json:"reported_at"`
}

var reportTiers = []ratelimit.Tier{
	{Limit: 5, Window: time.Minute},
	{Limit: 20, Window: time.Hour},
	{Limit: 100, Window: 24 * time.Hour},
}

func (h *Handlers) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req reportIn
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	if req.DrugID == 0 || req.PharmacyID == 0 {
		writeErr(w, 400, "drug_id and pharmacy_id required")
		return
	}
	switch req.Status {
	case "in_stock", "out", "limited":
	default:
		writeErr(w, 400, "status must be in_stock, out or limited")
		return
	}

	rh := ratelimit.ReporterHash(r)
	if !ratelimit.Burst(rh, reportTiers) {
		writeErr(w, 429, "slow down")
		return
	}

	ctx := r.Context()

	var pharmacyName string
	err := h.DB.QueryRow(ctx, "select name from pharmacies where id = $1", req.PharmacyID).Scan(&pharmacyName)
	if err != nil {
		writeErr(w, 404, "pharmacy not found")
		return
	}
	var one int
	if err := h.DB.QueryRow(ctx, "select 1 from drugs where id = $1", req.DrugID).Scan(&one); err != nil {
		writeErr(w, 404, "drug not found")
		return
	}

	var dup int
	_ = h.DB.QueryRow(ctx, `
        select 1 from reports
        where reporter_hash = $1 and drug_id = $2 and pharmacy_id = $3
          and reported_at > now() - interval '2 minutes'
        limit 1`, rh, req.DrugID, req.PharmacyID).Scan(&dup)
	if dup == 1 {
		writeErr(w, 409, "duplicate report")
		return
	}

	var price any
	if req.PricePaise > 0 {
		price = req.PricePaise
	}

	var out reportOut
	err = h.DB.QueryRow(ctx, `
        insert into reports (drug_id, pharmacy_id, status, price_paise, reporter_hash, note)
        values ($1,$2,$3,$4,$5,$6)
        returning id, drug_id, pharmacy_id, status, coalesce(note,''), reported_at`,
		req.DrugID, req.PharmacyID, req.Status, price, rh, req.Note,
	).Scan(&out.ID, &out.DrugID, &out.PharmacyID, &out.Status, &out.Note, &out.ReportedAt)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out.PharmacyName = pharmacyName
	writeJSON(w, 201, out)
}

func (h *Handlers) RecentReports(w http.ResponseWriter, r *http.Request) {
	drugID, _ := strconv.ParseInt(r.URL.Query().Get("drug_id"), 10, 64)
	if drugID == 0 {
		writeErr(w, 400, "drug_id required")
		return
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := h.DB.Query(r.Context(), `
        select r.id, r.drug_id, r.pharmacy_id, p.name, r.status, coalesce(r.note,''), r.reported_at
        from reports r join pharmacies p on p.id = r.pharmacy_id
        where r.drug_id = $1
        order by r.reported_at desc
        limit $2`, drugID, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := []reportOut{}
	for rows.Next() {
		var o reportOut
		if err := rows.Scan(&o.ID, &o.DrugID, &o.PharmacyID, &o.PharmacyName, &o.Status, &o.Note, &o.ReportedAt); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		out = append(out, o)
	}
	writeJSON(w, 200, out)
}
