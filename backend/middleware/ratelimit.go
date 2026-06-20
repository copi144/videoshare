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

// RateLimit returns middleware that limits requests to maxCount per window duration per IP,
// along with a stop function that halts the internal cleanup goroutine.
func RateLimit(maxCount int, window time.Duration, skip ...func(*http.Request) bool) (func(http.Handler) http.Handler, func()) {
	var mu sync.Mutex
	visits := make(map[string]*visitor)

	stop := make(chan struct{})

	// Cleanup goroutine removes stale entries.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for ip, v := range visits {
					if time.Since(v.lastSeen) > window {
						delete(visits, ip)
					}
				}
				mu.Unlock()
			case <-stop:
				return
			}
		}
	}()

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass rate limiting for requests that match any skip function (e.g. API token auth).
			if len(skip) > 0 {
				for _, fn := range skip {
					if fn(r) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

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

	stopFunc := func() { close(stop) }
	return mw, stopFunc
}
