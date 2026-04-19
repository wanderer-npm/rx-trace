package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *Handlers) NearbyPharmacies(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if lat == 0 || lng == 0 {
		writeErr(w, 400, "lat and lng required")
		return
	}
	radius, _ := strconv.ParseInt(r.URL.Query().Get("r"), 10, 64)
	if radius <= 0 || radius > 50000 {
		radius = 5000
	}
	limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := h.DB.Query(r.Context(), `
        select id, name, coalesce(address,''), coalesce(city,''), coalesce(phone,''),
               st_y(location::geometry), st_x(location::geometry),
               st_distance(location, st_makepoint($2,$1)::geography),
               verified
        from pharmacies
        where st_dwithin(location, st_makepoint($2,$1)::geography, $3)
        order by st_distance(location, st_makepoint($2,$1)::geography) asc
        limit $4`, lat, lng, radius, limit)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	defer rows.Close()

	out := []Pharmacy{}
	for rows.Next() {
		var p Pharmacy
		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.City, &p.Phone, &p.Lat, &p.Lng, &p.DistanceM, &p.Verified); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		out = append(out, p)
	}
	writeJSON(w, 200, out)
}

type addPharmacyReq struct {
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Address string  `json:"address"`
	City    string  `json:"city"`
	State   string  `json:"state"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

func (h *Handlers) AddPharmacy(w http.ResponseWriter, r *http.Request) {
	var req addPharmacyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	if req.Name == "" || req.Lat == 0 || req.Lng == 0 {
		writeErr(w, 400, "name, lat, lng required")
		return
	}
	country := req.Country
	if country == "" {
		country = "IN"
	}
	var p Pharmacy
	err := h.DB.QueryRow(r.Context(), `
        insert into pharmacies (name, phone, address, city, state, country, location)
        values ($1,$2,$3,$4,$5,$6, st_makepoint($8,$7)::geography)
        returning id, name, coalesce(address,''), coalesce(city,''), coalesce(phone,''),
                  st_y(location::geometry), st_x(location::geometry), verified`,
		req.Name, req.Phone, req.Address, req.City, req.State, country, req.Lat, req.Lng,
	).Scan(&p.ID, &p.Name, &p.Address, &p.City, &p.Phone, &p.Lat, &p.Lng, &p.Verified)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, p)
}
