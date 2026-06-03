package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	cfgPath string

	mu         sync.RWMutex
	cfg        Config
	results    map[string]Result
	httpClient *http.Client

	pollerCancel map[string]context.CancelFunc
	notifier *ChannelNotifier
}

func newServer(cfgPath string) (*Server, error) {
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	debugf("Config is loaded")
	notifier := (*ChannelNotifier)(nil)
	if len(cfg.Channels) > 0 {
		debugf("Channels detected in config")
		notifier = newChannelNotifier(cfg.Channels)
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
	}
	client := &http.Client{Transport: transport}
	s := &Server{
		cfgPath:      cfgPath,
		cfg:          cfg,
		results:      make(map[string]Result),
		httpClient:   client,
		pollerCancel: make(map[string]context.CancelFunc),
		notifier: notifier,
	}
	debugf("Server is created")
	for name := range cfg.Services {
		s.results[name] = Result{OK: false, Timestamp: time.Time{}}
	}
	return s, nil
}

func (s *Server) runPollers(ctx context.Context) {
	debugf("Running pollers")

	s.mu.RLock()
	services := make(map[string]ServiceConfig, len(s.cfg.Services))
	for k, v := range s.cfg.Services {
		services[k] = v
	}
	s.mu.RUnlock()

	for name, sc := range services {
		s.startPoller(name, sc, ctx)
	}
}

func (s *Server) startPoller(name string, sc ServiceConfig, parentCtx context.Context) {
	s.mu.Lock()
	if cancel, ok := s.pollerCancel[name]; ok {
		debugf("cancelling existing poller for %s", name)
		cancel()
		delete(s.pollerCancel, name)
	}
	ctx, cancel := context.WithCancel(parentCtx)
	s.pollerCancel[name] = cancel
	s.mu.Unlock()

	interval, err := parseInterval(sc.Interval)
	if err != nil || interval <= 0 {
		debugf("invalid interval for %s (%q), defaulting to 1m", name, sc.Interval)
		interval = time.Minute
	} else {
		debugf("starting poller for %s every %s", name, interval)
	}
	if (sc.MaintenancePeriod != nil) {
		m := *sc.MaintenancePeriod
		debugf(
			"Noting the maintenance period for %s: [Repeated %s, starting from %s %s and lasting for %s]",
			name,
			m.Repeat,
			m.StartingDay,
			m.StartingTime,
			m.Duration,
		)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				debugf("poller for %s stopped", name)
				return
			default:
			}
			now := time.Now().UTC()
			if inMaintenanceWindow(sc.MaintenancePeriod, now) {
				debugf("skipping polling %s during maintenance window", name)
				//TODO: set a "maintenance" result; otherwise leave previous state unchanged
			} else {
				debugf("polling %s -> %s", name, sc.FQDN)
				res := s.pingService(sc)
				res.Timestamp = now
				if res.OK != s.results[name].OK {
					s.mu.Lock()
					s.results[name] = res
					if s.notifier != nil && len(sc.Channels) > 0 {
						go s.notifier.NotifyForServiceState(name, res, sc.Channels)
					}
					s.mu.Unlock()
					infof("result for %s: ok=%v status=%d attempts=%d err=%q dur_ms=%d",
						name, res.OK, res.StatusCode, res.Attempts, res.Error, res.DurationMs)
				} else {
					infof("result same for %s [OK: %v]", name, res.OK)
				}
			}
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				debugf("poller for %s stopped (during sleep)", name)
				return
			}
		}
	}()
}
