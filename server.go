package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var healthy int32

func Serve(router *http.Handler, logger *log.Logger) {
    if logger == nil {
        logger = log.New(os.Stdout, "wacore: ", log.LstdFlags)
    }

	logger.Printf("server start\n")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      *router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	atomic.StoreInt32(&healthy, 1)
	go func() {
		<-quit
		logger.Printf("server is shutting down...\n")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("couldn't shutdown the server gracefully\n")
		}

		close(done)
	}()

	logger.Printf("server is running on: http://localhost%s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("couldn't listen and serve: %v\n", err.Error())
	}

	<-done
	logger.Printf("server shutdown\n")
}
