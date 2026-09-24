// Deterministic stub target for benchmarks. Fixed behavior, no randomness
// at runtime (flakiness derived from request count — reproducible).
package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var n int64

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(200)
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(200)
	})
	mux.HandleFunc("/flaky", func(w http.ResponseWriter, r *http.Request) {
		// Alternates 200/500 deterministically: every 2nd request fails.
		// With threshold 3, incidents open and resolve on a fixed cadence.
		if atomic.AddInt64(&n, 1)%2 == 0 {
			w.WriteHeader(500)
			fmt.Fprintln(w, "flaky")
			return
		}
		w.WriteHeader(200)
	})
	mux.HandleFunc("/timeout", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(12 * time.Second)
		w.WriteHeader(200)
	})
	s := &http.Server{Addr: ":8099", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	fmt.Println("stub listening on :8099")
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
