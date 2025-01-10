package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const requestIDKey = 0
var healthy int32

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requestID := r.Header.Get("X-Request-Id")
            if requestID == "" {
                requestID = nextRequestID()
            }
            ctx := context.WithValue(r.Context(), requestIDKey, requestID)
            w.Header().Set("X-Request-Id", requestID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
            defer func() {
                requestID, ok := r.Context().Value(requestIDKey).(string)
                if !ok {
                    requestID = "unknown"
                }
                logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
            }()
            next.ServeHTTP(w, r)
        })
    }
}

func main() {
    logger := log.New(os.Stdout, "wacore: ", log.LstdFlags)
    logger.Printf("start\n")

    http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("hello\nworld!\n"))
    })

    http.HandleFunc("GET /healtz", func(w http.ResponseWriter, r *http.Request) {
        if atomic.LoadInt32(&healthy) == 1 {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        w.WriteHeader(http.StatusServiceUnavailable)
    })

    nextRequestID := func() string {
        return strconv.FormatInt(time.Now().UnixNano(), 10)
    }

    server := http.Server{
        Addr:         ":8080",
        Handler:      tracing(nextRequestID)(logging(logger)(http.DefaultServeMux)),
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

        ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
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
    logger.Printf("server is shutdown\n")
}
