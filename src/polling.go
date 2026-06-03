package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Result struct {
	OK         bool      `json:"ok"`
	StatusCode int       `json:"status_code,omitempty"`
	Error      string    `json:"error,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	DurationMs int64     `json:"duration_ms"`
	Attempts   int       `json:"attempts"`
}

func (s *Server) pingService(sc ServiceConfig) Result {
	if sc.Method == "" {
		sc.Method = "curl"
	}
	if sc.Method != "curl" {
		return Result{
			OK:    false,
			Error: fmt.Sprintf("unsupported method: %s", sc.Method),
		}
	}
	timeout := time.Duration(sc.TimeoutS) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	maxAttempts := sc.Retries
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	var statusCode int
	var dur time.Duration
	attempts := 0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attempts = attempt
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sc.FQDN, nil)
		if err != nil {
			cancel()
			lastErr = err
			break
		}
		resp, err := s.httpClient.Do(req)
		dur = time.Since(start)
		cancel()
		if err != nil {
			lastErr = err
			debugf("request error for %s: %v (attempt %d/%d). Next retry in %d seconds...", sc.FQDN, err, attempt, maxAttempts, sc.RetryIntervalSeconds)
			time.Sleep(time.Duration(sc.RetryIntervalSeconds) * time.Second)
			continue
		}
		statusCode = resp.StatusCode
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if statusCode >= 200 && statusCode < 300 {
			return Result{
				OK:         true,
				StatusCode: statusCode,
				Timestamp:  time.Now().UTC(),
				DurationMs: dur.Milliseconds(),
				Attempts:   attempts,
			}
		}
		lastErr = errors.New(fmt.Sprintf("status %d", statusCode))
		debugf("non-2xx for %s: %d (attempt %d/%d). Next retry in %d seconds...", sc.FQDN, statusCode, attempt, maxAttempts, sc.RetryIntervalSeconds)
		time.Sleep(time.Duration(sc.RetryIntervalSeconds) * time.Second)
	}
	errStr := ""
	if lastErr != nil {
		errStr = lastErr.Error()
	}
	return Result{
		OK:         false,
		StatusCode: statusCode,
		Error:      errStr,
		Timestamp:  time.Now().UTC(),
		DurationMs: dur.Milliseconds(),
		Attempts:   attempts,
	}
}
