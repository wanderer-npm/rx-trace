package handlers

import (
	"net/http"
	"strconv"
	"time"
)

type FindRow struct {
	Pharmacy      Pharmacy  `json:"pharmacy"`
	Status        string    `json:"status"`
	ReportedAt    time.Time `json:"reported_at"`
	FreshnessMin  int       `json:"freshness_min"`
}

func (h *Handlers) Find(w http.ResponseWriter, r *http.Request) {
	drugID, _ := strconv.ParseInt(r.URL.Query().Get("drug_id"), 10, 64)
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if drugID == 0 || lat == 0 || lng == 0 {
		writeErr(w, 400, "drug_id, lat, lng required")
		return
	}
	radius, _ := strconv.ParseInt(r.URL.Query().Get("radius_m"), 10, 64)
	if radius <= 0 || radius > 25000 {
		radius = 5000
	}
	includeStale := r.URL.Query().Get("include_stale") == "true"
	maxAge, _ := strconv.ParseInt(r.URL.Query().Get("max_age_hours"), 10, 64)
	if maxAge <= 0 || maxAge > 720 {
		maxAge = 48
	}

	rows, err := h.DB.Query(r.Context(), `
        with latest as (
            select distinct on (r.pharmacy_id)
                r.pharmacy_id, r.status, r.reported_at
            from reports r
            where r.drug_id = $1
            order by r.pharmacy_id, r.reported_at desc
        )
        select p.id, p.name,
               coalesce(p.address,''), coalesce(p.city,''), coalesce(p.phone,''),
               st_y(p.location::geometry), st_x(p.location::geometry),
               st_distance(p.location, st_makepoint($3, $2)::geography),
               p.verified,
               l.status, l.reported_at,
               (extract(epoch from (now() - l.reported_at)) / 60)::int as freshness_min
        from latest l
        join pharmacies p on p.id = l.pharmacy_id
        where st_dwithin(p.location, st_makepoint($3, $2)::geography, $4)
          and ($5 or l.status in ('in_stock', 'limited'))
          and l.reported_at > now() - make_interval(hours => $6)
        order by
            case l.status when 'in_stock' then 0 when 'limited' then 1 else 2 end,
            st_distance(p.location, st_makepoint($3, $2)::geography) asc
        limit 100`,
		drugID, lat, lng, radius, includeStale, maxAge)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := []FindRow{}
	for rows.Next() {
		var row FindRow
		if err := rows.Scan(
			&row.Pharmacy.ID, &row.Pharmacy.Name,
			&row.Pharmacy.Address, &row.Pharmacy.City, &row.Pharmacy.Phone,
			&row.Pharmacy.Lat, &row.Pharmacy.Lng,
			&row.Pharmacy.DistanceM, &row.Pharmacy.Verified,
			&row.Status, &row.ReportedAt, &row.FreshnessMin,
		); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		out = append(out, row)
	}
	writeJSON(w, 200, out)
}
