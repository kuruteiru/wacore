package main

import (
    "net/http"
    "sync/atomic"
)

func Route() http.Handler {
    router := http.NewServeMux()

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello\nworld!\n"))
	})

	router.HandleFunc("GET /healtz", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

    return router
}
