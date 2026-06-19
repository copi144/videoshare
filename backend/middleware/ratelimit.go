package middleware

import (
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

// RateLimit returns middleware that limits requests to maxCount per window duration per IP.
func RateLimit(maxCount int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	visits := make(map[string]*visitor)

	// Cleanup goroutine removes stale entries.
	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			for ip, v := range visits {
				if time.Since(v.lastSeen) > window {
					delete(visits, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			mu.Lock()
			v, exists := visits[ip]
			if !exists {
				visits[ip] = &visitor{count: 1, lastSeen: time.Now()}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Reset if the window has passed.
			if time.Since(v.lastSeen) > window {
				v.count = 1
				v.lastSeen = time.Now()
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			v.count++
			v.lastSeen = time.Now()

			if v.count > maxCount {
				mu.Unlock()
				http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
				return
			}

			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
