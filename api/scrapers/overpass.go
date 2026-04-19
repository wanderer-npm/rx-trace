package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var overpassEndpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
	"https://overpass.openstreetmap.fr/api/interpreter",
}

type BBox struct{ South, West, North, East float64 }

var CityBBoxes = map[string]BBox{
	"delhi":     {28.40, 76.84, 28.88, 77.35},
	"mumbai":    {18.89, 72.77, 19.27, 72.99},
	"bengaluru": {12.83, 77.46, 13.14, 77.78},
	"hyderabad": {17.27, 78.26, 17.55, 78.63},
	"chennai":   {12.83, 80.12, 13.23, 80.32},
	"kolkata":   {22.43, 88.23, 22.67, 88.49},
	"pune":      {18.43, 73.72, 18.63, 73.99},
	"ahmedabad": {22.94, 72.43, 23.18, 72.72},
	"jaipur":    {26.79, 75.70, 27.02, 75.92},
	"lucknow":   {26.72, 80.87, 26.95, 81.06},
}

func buildQuery(b BBox) string {
	return fmt.Sprintf(`[out:json][timeout:60];
(
  node["amenity"="pharmacy"](%f,%f,%f,%f);
  way["amenity"="pharmacy"](%f,%f,%f,%f);
);
out center tags;`, b.South, b.West, b.North, b.East, b.South, b.West, b.North, b.East)
}

func fetchOverpass(ctx context.Context, query string) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	form := url.Values{"data": {query}}
	var lastErr error
	for _, ep := range overpassEndpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, strings.NewReader(form.Encode()))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "rx-trace/0.1 (github.com/wanderer-npm/rx-trace)")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			slog.Warn("overpass request failed", "endpoint", ep, "err", err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 200 {
			return body, nil
		}
		lastErr = fmt.Errorf("%s returned %d", ep, resp.StatusCode)
		slog.Warn("overpass non-200", "endpoint", ep, "status", resp.StatusCode)
	}
	return nil, lastErr
}

func joinAddress(t map[string]string) string {
	parts := []string{t["addr:housenumber"], t["addr:street"], t["addr:suburb"], t["addr:neighbourhood"]}
	out := strings.TrimSpace(strings.Join(parts, " "))
	return strings.Join(strings.Fields(out), " ")
}

func OverpassImport(ctx context.Context, pool *pgxpool.Pool, b BBox) (int, error) {
	body, err := fetchOverpass(ctx, buildQuery(b))
	if err != nil {
		return 0, err
	}
	var env overpassEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return 0, err
	}

	inserted := 0
	for _, el := range env.Elements {
		tags := el.Tags
		name := tags["name"]
		if name == "" {
			name = tags["brand"]
		}
		if name == "" {
			name = tags["operator"]
		}
		if name == "" {
			continue
		}

		lat, lng := el.Lat, el.Lon
		if el.Type != "node" && el.Center != nil {
			lat, lng = el.Center.Lat, el.Center.Lon
		}
		if lat == 0 && lng == 0 {
			continue
		}

		address := joinAddress(tags)
		city := tags["addr:city"]
		if city == "" {
			city = tags["addr:district"]
		}
		phone := tags["phone"]
		if phone == "" {
			phone = tags["contact:phone"]
		}

		var hoursJSON *string
		if raw, ok := tags["opening_hours"]; ok && raw != "" {
			j, _ := json.Marshal(map[string]string{"raw": raw})
			s := string(j)
			hoursJSON = &s
		}

		if len(name) > 200 {
			name = name[:200]
		}

		_, err := pool.Exec(ctx, `
            insert into pharmacies (osm_id, name, phone, address, city, country, location, hours)
            values ($1, $2, $3, $4, $5, 'IN', st_makepoint($7, $6)::geography, $8::jsonb)
            on conflict (osm_id) do update set
                name = excluded.name,
                phone = coalesce(excluded.phone, pharmacies.phone),
                address = coalesce(excluded.address, pharmacies.address),
                city = coalesce(excluded.city, pharmacies.city),
                hours = coalesce(excluded.hours, pharmacies.hours)`,
			el.ID, name, nilIfEmpty(phone), nilIfEmpty(address), nilIfEmpty(city), lat, lng, hoursJSON)
		if err != nil {
			slog.Warn("pharmacy insert failed", "osm_id", el.ID, "err", err)
			continue
		}
		inserted++
	}

	_, _ = pool.Exec(ctx, `
        insert into sources (key, last_run_at, last_status, records_seen)
        values ('overpass', now(), 'ok', $1)
        on conflict (key) do update set
            last_run_at = excluded.last_run_at,
            last_status = 'ok',
            records_seen = excluded.records_seen`, inserted)

	return inserted, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
