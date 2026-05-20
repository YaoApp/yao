package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := flag.Int("port", 9999, "listen port")
	fixtureDir := flag.String("fixtures", "./fixtures", "fixture directory path")
	flag.Parse()

	if err := LoadFixtures(*fixtureDir); err != nil {
		log.Printf("WARN: failed to load fixtures: %v (fixture mode disabled)", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleOpenAI)
	mux.HandleFunc("/v1/messages", handleAnthropic)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("mock-llm listening on %s (fixtures=%s)", addr, *fixtureDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
		os.Exit(1)
	}
}
