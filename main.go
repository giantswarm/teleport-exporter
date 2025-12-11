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
)

func main() {
	var (
		metricsAddr     string
		probeAddr       string
		teleportAddr    string
		identityFile    string
		refreshInterval time.Duration
		insecure        bool
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&teleportAddr, "teleport-addr", "", "The address of the Teleport proxy/auth server (e.g., teleport.example.com:443).")
	flag.StringVar(&identityFile, "identity-file", "", "Path to the identity file for authentication.")
	flag.DurationVar(&refreshInterval, "refresh-interval", 30*time.Second, "How often to refresh metrics from Teleport API.")
	flag.BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification (not recommended for production).")
	flag.Parse()

	// Initialize logger
	zapLog, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync()
	log := zapr.NewLogger(zapLog)

	if teleportAddr == "" {
		log.Error(nil, "teleport-addr is required")
		os.Exit(1)
	}

	if identityFile == "" {
		log.Error(nil, "identity-file is required")
		os.Exit(1)
	}

	log.Info("Starting teleport-exporter",
		"teleportAddr", teleportAddr,
		"metricsAddr", metricsAddr,
		"refreshInterval", refreshInterval,
	)

	// Create Teleport client
	teleportClient, err := teleport.NewClient(teleport.Config{
		ProxyAddr:    teleportAddr,
		IdentityFile: identityFile,
		Insecure:     insecure,
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
		Log:             log.WithName("collector"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the collector
	go col.Run(ctx)

	// Set up metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: metricsMux,
	}

	// Set up health probe server
	probeMux := http.NewServeMux()
	probeMux.HandleFunc("/healthz", healthHandler)
	probeMux.HandleFunc("/readyz", readyHandler(teleportClient))

	probeServer := &http.Server{
		Addr:    probeAddr,
		Handler: probeMux,
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
	<-sigCh

	log.Info("shutting down")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(err, "failed to shutdown metrics server")
	}
	if err := probeServer.Shutdown(shutdownCtx); err != nil {
		log.Error(err, "failed to shutdown probe server")
	}
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
