package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// parseInterval accepts "@1m", "@5h", "@1d", or explicit durations like "30s".
func parseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty interval")
	}
	if strings.HasPrefix(s, "@") {
		v := strings.TrimPrefix(s, "@")
		if strings.HasSuffix(v, "m") || strings.HasSuffix(v, "h") || strings.HasSuffix(v, "d") {
			unit := v[len(v)-1]
			num := v[:len(v)-1]
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, err
			}
			switch unit {
			case 'm':
				return time.Duration(n) * time.Minute, nil
			case 'h':
				return time.Duration(n) * time.Hour, nil
			case 'd':
				return time.Duration(n) * 24 * time.Hour, nil
			default:
				return 0, fmt.Errorf("unknown unit: %c", unit)
			}
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Second, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid interval: %s", s)
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
			debugf("request error for %s: %v (attempt %d/%d)", sc.FQDN, err, attempt, maxAttempts)
			time.Sleep(time.Duration(500*attempt) * time.Millisecond)
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
		debugf("non-2xx for %s: %d (attempt %d/%d)", sc.FQDN, statusCode, attempt, maxAttempts)
		time.Sleep(time.Duration(500*attempt) * time.Millisecond)
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
