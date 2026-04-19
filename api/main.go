package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/wanderer-npm/rx-trace/api/db"
	"github.com/wanderer-npm/rx-trace/api/handlers"
	"github.com/wanderer-npm/rx-trace/api/scrapers"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scrape":
			os.Exit(scrapeCommand(os.Args[2:]))
		case "dispatch-alerts":
			os.Exit(dispatchCommand())
		}
	}

	serve()
}

func serve() {
	ctx := context.Background()
	pool, err := db.Connect(ctx)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	h := handlers.New(pool)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/drugs", h.SearchDrugs)
		r.Get("/drugs/{id}", h.GetDrug)

		r.Get("/pharmacies/nearby", h.NearbyPharmacies)
		r.Post("/pharmacies", h.AddPharmacy)

		r.Post("/reports", h.CreateReport)
		r.Get("/reports/recent", h.RecentReports)

		r.Get("/find", h.Find)
		r.Get("/shortages", h.ActiveShortages)

		r.Post("/alerts", h.CreateAlert)
		r.Get("/alerts", h.ListAlerts)
		r.Delete("/alerts/{id}", h.DeleteAlert)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("api listening", "port", port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("serve", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	shutdown, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func scrapeCommand(args []string) int {
	fs := flag.NewFlagSet("scrape", flag.ExitOnError)
	source := fs.String("source", "", "fda | overpass")
	city := fs.String("city", "", "built-in city for overpass (one of: "+strings.Join(cities(), ", ")+")")
	bbox := fs.String("bbox", "", "custom bbox for overpass as s,w,n,e")
	_ = fs.Parse(args)

	if *source == "" {
		fmt.Fprintln(os.Stderr, "usage: rx-trace scrape --source fda|overpass [--city delhi | --bbox s,w,n,e]")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx)
	if err != nil {
		slog.Error("db connect", "err", err)
		return 1
	}
	defer pool.Close()

	switch *source {
	case "fda":
		n, err := scrapers.FDAImport(ctx, pool)
		if err != nil {
			slog.Error("fda import", "err", err)
			return 1
		}
		slog.Info("fda done", "records", n)
	case "overpass":
		var b scrapers.BBox
		if *city != "" {
			bb, ok := scrapers.CityBBoxes[strings.ToLower(*city)]
			if !ok {
				fmt.Fprintf(os.Stderr, "unknown city %q (known: %s)\n", *city, strings.Join(cities(), ", "))
				return 2
			}
			b = bb
		} else if *bbox != "" {
			if _, err := fmt.Sscanf(*bbox, "%f,%f,%f,%f", &b.South, &b.West, &b.North, &b.East); err != nil {
				fmt.Fprintln(os.Stderr, "bbox must be s,w,n,e")
				return 2
			}
		} else {
			fmt.Fprintln(os.Stderr, "overpass: provide --city or --bbox")
			return 2
		}
		n, err := scrapers.OverpassImport(ctx, pool, b)
		if err != nil {
			slog.Error("overpass import", "err", err)
			return 1
		}
		slog.Info("overpass done", "pharmacies", n)
	default:
		fmt.Fprintf(os.Stderr, "unknown source %q\n", *source)
		return 2
	}
	return 0
}

func cities() []string {
	out := make([]string, 0, len(scrapers.CityBBoxes))
	for k := range scrapers.CityBBoxes {
		out = append(out, k)
	}
	return out
}
