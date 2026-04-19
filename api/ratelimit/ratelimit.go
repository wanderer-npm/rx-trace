package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Tier struct {
	Limit  int
	Window time.Duration
}

var (
	mu   sync.Mutex
	hits = make(map[string][]time.Time)
)

func ReporterHash(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		if i := strings.Index(ip, ","); i >= 0 {
			ip = ip[:i]
		}
		ip = strings.TrimSpace(ip)
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	ua := r.Header.Get("User-Agent")
	sum := sha256.Sum256([]byte(ip + "|" + ua))
	return hex.EncodeToString(sum[:])[:32]
}

func Allow(key string, limit int, window time.Duration) bool {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-window)
	xs := hits[key]
	fresh := xs[:0]
	for _, t := range xs {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= limit {
		hits[key] = fresh
		return false
	}
	fresh = append(fresh, now)
	hits[key] = fresh
	return true
}

func Burst(key string, tiers []Tier) bool {
	for _, t := range tiers {
		if !Allow(key+"|"+t.Window.String(), t.Limit, t.Window) {
			return false
		}
	}
	return true
}
