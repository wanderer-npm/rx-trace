package scrapers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const fdaEndpoint = "https://api.fda.gov/drug/drugshortages.json?limit=1000"

func parseDate(s string) any {
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", "01/02/2006", "20060102"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return nil
}

func FDAImport(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fdaEndpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "rx-trace/0.1 (github.com/wanderer-npm/rx-trace)")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var env fdaEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return 0, err
	}

	inserted := 0
	for _, rec := range env.Results {
		name := rec.ProprietaryName
		generic := rec.GenericName
		if name == "" && len(rec.OpenFDA.BrandName) > 0 {
			name = rec.OpenFDA.BrandName[0]
		}
		if name == "" {
			name = generic
		}
		if name == "" {
			continue
		}
		if len(name) > 200 {
			name = name[:200]
		}
		if len(generic) > 200 {
			generic = generic[:200]
		}

		var drugID int64
		err := pool.QueryRow(ctx, `
            insert into drugs (name, generic)
            values ($1, nullif($2, ''))
            on conflict (name, form, strength) do update set
                generic = coalesce(excluded.generic, drugs.generic)
            returning id`, name, generic).Scan(&drugID)
		if err != nil {
			slog.Warn("drug upsert failed", "name", name, "err", err)
			continue
		}

		raw, _ := json.Marshal(map[string]string{
			"reason":       rec.Reason,
			"availability": rec.Availability,
			"update_type":  rec.UpdateType,
			"update_date":  rec.UpdateDate,
		})
		_, err = pool.Exec(ctx, `
            insert into shortages (drug_id, source, severity, started_at, expected_end, url, raw)
            values ($1, 'fda', $2, $3, null, null, $4::jsonb)
            on conflict (drug_id, source, started_at) do update set
                severity = excluded.severity,
                raw = excluded.raw`,
			drugID, rec.Status, parseDate(rec.InitialPostingDate), string(raw))
		if err != nil {
			slog.Warn("shortage upsert failed", "drug_id", drugID, "err", err)
			continue
		}
		inserted++
	}

	_, _ = pool.Exec(ctx, `
        insert into sources (key, last_run_at, last_status, records_seen)
        values ('fda', now(), 'ok', $1)
        on conflict (key) do update set
            last_run_at = excluded.last_run_at,
            last_status = 'ok',
            records_seen = excluded.records_seen`, inserted)

	return inserted, nil
}
