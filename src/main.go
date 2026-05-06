package main

import (
	"context"
	"net/http"
	"os"
)

func main() {
	cfgPath := os.Getenv("POLLIO_SERVICES_FILE")
	if cfgPath == "" {
		errorf("POLLIO_SERVICES_FILE not set")
	}
	addr := "0.0.0.0:8080"

	srv, err := newServer(cfgPath)
	if err != nil {
		errorf("failed to start: %v\n", err)
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.runPollers(rootCtx)
	http.HandleFunc("/status", srv.handleStatus)

	httpServer := &http.Server{
		Addr: addr,
	}

	debugf("listening on %s", addr)
	infof("Server is ready!")
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		warningf("http server error: %v\n", err)
	}
}
