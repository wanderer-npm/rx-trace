package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wanderer-npm/rx-trace/api/db"
	"github.com/wanderer-npm/rx-trace/api/email"
)

func dispatchCommand() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx)
	if err != nil {
		slog.Error("db connect", "err", err)
		return 1
	}
	defer pool.Close()

	sent, err := dispatchAlerts(ctx, pool)
	if err != nil {
		slog.Error("dispatch", "err", err)
		return 1
	}
	slog.Info("dispatch done", "sent", sent)
	return 0
}

func dispatchAlerts(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	rows, err := pool.Query(ctx, `
        select a.id, a.drug_id, d.name as drug_name,
               a.lat, a.lng, a.radius_m, a.channel, a.target,
               coalesce(a.last_fired_at, now() - interval '30 days') as since
        from alerts a
        join drugs d on d.id = a.drug_id
        where a.active = true
        order by a.id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type row struct {
		id       int64
		drugID   int64
		drugName string
		lat, lng float64
		radius   int
		channel  string
		target   string
		since    time.Time
	}
	var alerts []row
	for rows.Next() {
		var a row
		if err := rows.Scan(&a.id, &a.drugID, &a.drugName, &a.lat, &a.lng, &a.radius, &a.channel, &a.target, &a.since); err != nil {
			return 0, err
		}
		alerts = append(alerts, a)
	}

	sent := 0
	siteBase := os.Getenv("SITE_URL")
	if siteBase == "" {
		siteBase = "http://localhost:3000"
	}

	for _, a := range alerts {
		hitsRows, err := pool.Query(ctx, `
            with latest as (
                select distinct on (r.pharmacy_id)
                    r.pharmacy_id, r.status, r.reported_at
                from reports r
                where r.drug_id = $1 and r.reported_at > $2
                order by r.pharmacy_id, r.reported_at desc
            )
            select p.name, coalesce(p.address,''), coalesce(p.city,''),
                   st_distance(p.location, st_makepoint($4, $3)::geography) as dist
            from latest l
            join pharmacies p on p.id = l.pharmacy_id
            where st_dwithin(p.location, st_makepoint($4, $3)::geography, $5)
              and l.status = 'in_stock'
            order by dist asc
            limit 10`, a.drugID, a.since, a.lat, a.lng, a.radius)
		if err != nil {
			slog.Warn("alert query failed", "id", a.id, "err", err)
			continue
		}

		var hits []email.StockHit
		for hitsRows.Next() {
			var h email.StockHit
			var city string
			if err := hitsRows.Scan(&h.Pharmacy, &h.Address, &city, &h.DistanceM); err != nil {
				continue
			}
			if city != "" {
				if h.Address == "" {
					h.Address = city
				} else {
					h.Address = h.Address + ", " + city
				}
			}
			hits = append(hits, h)
		}
		hitsRows.Close()

		if len(hits) == 0 {
			continue
		}

		unsub := fmt.Sprintf("%s/unsubscribe?id=%d&target=%s", siteBase, a.id, a.target)

		if a.channel == "email" {
			if err := email.SendStockAlert(a.target, a.drugName, hits, unsub); err != nil {
				slog.Warn("email send failed", "id", a.id, "err", err)
				continue
			}
			sent++
			_, _ = pool.Exec(ctx, "update alerts set last_fired_at = now() where id = $1", a.id)
		}
	}
	return sent, nil
}
