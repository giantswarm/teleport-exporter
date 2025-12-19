/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/zapr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/giantswarm/teleport-exporter/internal/collector"
	"github.com/giantswarm/teleport-exporter/internal/teleport"
	"github.com/giantswarm/teleport-exporter/internal/version"
)

const (
	// HTTP server timeouts for security hardening
	httpReadTimeout     = 10 * time.Second
	httpWriteTimeout    = 10 * time.Second
	httpIdleTimeout     = 60 * time.Second
	httpMaxHeaderBytes  = 1 << 20 // 1 MB
	httpShutdownTimeout = 10 * time.Second
)

func main() {
	var (
		metricsAddr     string
		probeAddr       string
		teleportAddr    string
		identityFile    string
		refreshInterval time.Duration
		apiTimeout      time.Duration
		insecure        bool
		showVersion     bool
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&teleportAddr, "teleport-addr", "", "The address of the Teleport proxy/auth server (e.g., teleport.example.com:443).")
	flag.StringVar(&identityFile, "identity-file", "", "Path to the identity file for authentication.")
	flag.DurationVar(&refreshInterval, "refresh-interval", 60*time.Second, "How often to refresh metrics from Teleport API.")
	flag.DurationVar(&apiTimeout, "api-timeout", 30*time.Second, "Timeout for Teleport API calls.")
	flag.BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification (not recommended for production).")
	flag.BoolVar(&showVersion, "version", false, "Print version information and exit.")
	flag.Parse()

	// Handle version flag
	if showVersion {
		v := version.Get()
		fmt.Printf("teleport-exporter %s\n", v.Version)
		fmt.Printf("  commit: %s\n", v.Commit)
		fmt.Printf("  built:  %s\n", v.BuildDate)
		fmt.Printf("  go:     %s\n", v.GoVersion)
		fmt.Printf("  platform: %s\n", v.Platform)
		os.Exit(0)
	}

	// Initialize logger
	zapLog, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync()
	log := zapr.NewLogger(zapLog)

	// Log version information at startup (no build_info metric to reduce cardinality)
	v := version.Get()
	log.Info("Starting teleport-exporter",
		"version", v.Version,
		"commit", v.Commit,
		"buildDate", v.BuildDate,
		"goVersion", v.GoVersion,
	)

	if teleportAddr == "" {
		log.Error(nil, "teleport-addr is required")
		os.Exit(1)
	}

	if identityFile == "" {
		log.Error(nil, "identity-file is required")
		os.Exit(1)
	}

	log.Info("Configuration",
		"teleportAddr", teleportAddr,
		"metricsAddr", metricsAddr,
		"probeAddr", probeAddr,
		"refreshInterval", refreshInterval,
		"apiTimeout", apiTimeout,
	)

	// Create Teleport client
	teleportClient, err := teleport.NewClient(teleport.Config{
		ProxyAddr:    teleportAddr,
		IdentityFile: identityFile,
		Insecure:     insecure,
		APITimeout:   apiTimeout,
		Log:          log.WithName("teleport-client"),
	})
	if err != nil {
		log.Error(err, "failed to create Teleport client")
		os.Exit(1)
	}
	defer teleportClient.Close()

	// Create and start the collector
	col := collector.New(collector.Config{
		TeleportClient:  teleportClient,
		RefreshInterval: refreshInterval,
		APITimeout:      apiTimeout,
		Log:             log.WithName("collector"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the collector
	go col.Run(ctx)

	// Set up metrics server with security hardening
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:           metricsAddr,
		Handler:        metricsMux,
		ReadTimeout:    httpReadTimeout,
		WriteTimeout:   httpWriteTimeout,
		IdleTimeout:    httpIdleTimeout,
		MaxHeaderBytes: httpMaxHeaderBytes,
	}

	// Set up health probe server with security hardening
	probeMux := http.NewServeMux()
	probeMux.HandleFunc("/healthz", healthHandler)
	probeMux.HandleFunc("/readyz", readyHandler(teleportClient))

	probeServer := &http.Server{
		Addr:           probeAddr,
		Handler:        probeMux,
		ReadTimeout:    httpReadTimeout,
		WriteTimeout:   httpWriteTimeout,
		IdleTimeout:    httpIdleTimeout,
		MaxHeaderBytes: httpMaxHeaderBytes,
	}

	// Start servers
	go func() {
		log.Info("starting metrics server", "addr", metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "metrics server failed")
		}
	}()

	go func() {
		log.Info("starting health probe server", "addr", probeAddr)
		if err := probeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "health probe server failed")
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Info("received shutdown signal", "signal", sig.String())
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer shutdownCancel()

	// Shutdown servers gracefully
	var shutdownErr error
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(err, "failed to shutdown metrics server")
		shutdownErr = err
	}
	if err := probeServer.Shutdown(shutdownCtx); err != nil {
		log.Error(err, "failed to shutdown probe server")
		shutdownErr = err
	}

	if shutdownErr != nil {
		log.Info("shutdown completed with errors")
		os.Exit(1)
	}
	log.Info("shutdown completed successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func readyHandler(client *teleport.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if client.IsConnected() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not connected to Teleport"))
		}
	}
}
