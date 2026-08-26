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

package teleport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/gravitational/teleport/api/client"
)

const (
	// Default timeout for API operations if not specified
	defaultAPITimeout = 30 * time.Second
	// identityReloadTimeout bounds a single identity file read, see ReloadIdentity.
	identityReloadTimeout = 1 * time.Second
	// healthCheckTimeout bounds the health check ping. Together with
	// identityReloadTimeout it has to stay inside the readiness probe's
	// timeoutSeconds (see helm/teleport-exporter/templates/deployment.yaml),
	// so that a slow check makes the handler report not ready instead of
	// making the kubelet time the probe out.
	healthCheckTimeout = 3 * time.Second
)

// Config holds the configuration for the Teleport client.
type Config struct {
	// ProxyAddr is the address of the Teleport proxy or auth server.
	ProxyAddr string
	// IdentityFile is the path to the identity file for authentication.
	IdentityFile string
	// Insecure skips TLS certificate verification.
	Insecure bool
	// APITimeout is the timeout for API calls.
	APITimeout time.Duration
	// Log is the logger to use.
	Log logr.Logger
}

// Client wraps the Teleport API client.
type Client struct {
	client     *client.Client
	creds      *client.DynamicIdentityFileCreds
	log        logr.Logger
	apiTimeout time.Duration
	connected  bool
	reloading  atomic.Bool
	mu         sync.RWMutex
}

// NodeInfo represents information about a Teleport node.
type NodeInfo struct {
	Name      string
	Hostname  string
	Address   string
	Labels    map[string]string
	Namespace string
	SubKind   string
}

// KubeClusterInfo represents information about a Kubernetes cluster registered in Teleport.
type KubeClusterInfo struct {
	Name   string
	Labels map[string]string
}

// DatabaseInfo represents information about a database registered in Teleport.
type DatabaseInfo struct {
	Name     string
	Protocol string
	Type     string
	Labels   map[string]string
}

// AppInfo represents information about an application registered in Teleport.
type AppInfo struct {
	Name       string
	PublicAddr string
	URI        string
	Labels     map[string]string
}

// NewClient creates a new Teleport client.
func NewClient(cfg Config) (*Client, error) {
	cfg.Log.Info("connecting to Teleport", "addr", cfg.ProxyAddr)

	apiTimeout := cfg.APITimeout
	if apiTimeout == 0 {
		apiTimeout = defaultAPITimeout
	}

	// Use timeout for initial connection
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()

	// client.LoadIdentityFile caches the file contents for the lifetime of the
	// credential, but tbot rotates the short lived Machine ID certificate on
	// disk, so it has to be served through callbacks instead - see
	// ReloadIdentity. These credentials authenticate SSH as the
	// -teleport-internal-join principal, which Teleport includes on every user
	// certificate.
	creds, err := client.NewDynamicIdentityFileCreds(cfg.IdentityFile)
	if err != nil {
		return nil, err
	}

	c, err := client.New(ctx, client.Config{
		Addrs:                    []string{cfg.ProxyAddr},
		Credentials:              []client.Credentials{creds},
		InsecureAddressDiscovery: cfg.Insecure,
	})
	if err != nil {
		return nil, err
	}

	cfg.Log.Info("connected to Teleport successfully")

	return &Client{
		client:     c,
		creds:      creds,
		log:        cfg.Log,
		apiTimeout: apiTimeout,
		connected:  true,
	}, nil
}

// ReloadIdentity re-reads the identity file from disk so that subsequent TLS
// and SSH handshakes use the current certificate. Without it the client keeps
// presenting the certificate that was on disk at startup and every reconnect
// eventually fails with "cert has expired". DynamicIdentityFileCreds
// deliberately does not reload on its own, so callers have to do it.
//
// The read is bounded, and skipped while an earlier one is still running: it
// is not context aware, so a stalled identity volume would otherwise hang
// every caller and, on the readiness path, leave a goroutine behind per probe.
// Both cases report an error even though the credential in place may still be
// current - a reload takes well under a millisecond, so anything slow enough
// to hit them is worth seeing.
func (c *Client) ReloadIdentity() error {
	if !c.reloading.CompareAndSwap(false, true) {
		return errors.New("previous identity file reload is still running")
	}

	done := make(chan error, 1)
	go func() {
		err := c.creds.Reload()
		c.reloading.Store(false)
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(identityReloadTimeout):
		return fmt.Errorf("identity file reload did not finish within %s", identityReloadTimeout)
	}
}

// Close closes the Teleport client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return c.client.Close()
}

// IsConnected returns whether the client is connected by performing a health check.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return false
	}
	c.mu.RUnlock()

	// Reload before pinging. The collector reloads on its own cadence too, but
	// that interval stretches by up to 256x under backoff, so the readiness
	// probe is what bounds how long a stale certificate can keep the client
	// unhealthy after the identity file has been rotated.
	if err := c.ReloadIdentity(); err != nil {
		c.log.V(1).Info("failed to reload identity file", "error", err)
	}

	// Perform actual health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := c.client.Ping(ctx)
	if err != nil {
		c.log.V(1).Info("health check failed", "error", err)
		return false
	}
	return true
}

// withTimeout returns a context with the configured API timeout.
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.apiTimeout)
}

// GetNodes returns all nodes registered in Teleport.
func (c *Client) GetNodes(ctx context.Context) ([]NodeInfo, error) {
	c.log.V(1).Info("fetching nodes from Teleport")

	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	nodes, err := c.client.GetNodes(ctx, "default")
	if err != nil {
		c.log.Error(err, "failed to get nodes")
		return nil, err
	}

	result := make([]NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, NodeInfo{
			Name:      node.GetName(),
			Hostname:  node.GetHostname(),
			Address:   node.GetAddr(),
			Labels:    node.GetAllLabels(),
			Namespace: node.GetNamespace(),
			SubKind:   node.GetSubKind(),
		})
	}

	c.log.V(1).Info("fetched nodes", "count", len(result))
	return result, nil
}

// GetKubeClusters returns all Kubernetes clusters registered in Teleport.
func (c *Client) GetKubeClusters(ctx context.Context) ([]KubeClusterInfo, error) {
	c.log.V(1).Info("fetching Kubernetes clusters from Teleport")

	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	clusters, err := c.client.GetKubernetesServers(ctx)
	if err != nil {
		c.log.Error(err, "failed to get Kubernetes clusters")
		return nil, err
	}

	// Use a map to deduplicate clusters (multiple servers can serve the same cluster)
	clusterMap := make(map[string]KubeClusterInfo)
	for _, server := range clusters {
		cluster := server.GetCluster()
		if cluster != nil {
			clusterMap[cluster.GetName()] = KubeClusterInfo{
				Name:   cluster.GetName(),
				Labels: cluster.GetAllLabels(),
			}
		}
	}

	result := make([]KubeClusterInfo, 0, len(clusterMap))
	for _, cluster := range clusterMap {
		result = append(result, cluster)
	}

	c.log.V(1).Info("fetched Kubernetes clusters", "count", len(result))
	return result, nil
}

// GetDatabases returns all databases registered in Teleport.
func (c *Client) GetDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	c.log.V(1).Info("fetching databases from Teleport")

	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	databases, err := c.client.GetDatabaseServers(ctx, "default")
	if err != nil {
		c.log.Error(err, "failed to get databases")
		return nil, err
	}

	// Use a map to deduplicate databases (multiple servers can serve the same database)
	dbMap := make(map[string]DatabaseInfo)
	for _, server := range databases {
		db := server.GetDatabase()
		if db != nil {
			dbMap[db.GetName()] = DatabaseInfo{
				Name:     db.GetName(),
				Protocol: db.GetProtocol(),
				Type:     db.GetType(),
				Labels:   db.GetAllLabels(),
			}
		}
	}

	result := make([]DatabaseInfo, 0, len(dbMap))
	for _, db := range dbMap {
		result = append(result, db)
	}

	c.log.V(1).Info("fetched databases", "count", len(result))
	return result, nil
}

// GetApps returns all applications registered in Teleport.
func (c *Client) GetApps(ctx context.Context) ([]AppInfo, error) {
	c.log.V(1).Info("fetching applications from Teleport")

	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	servers, err := c.client.GetApplicationServers(ctx, "default")
	if err != nil {
		c.log.Error(err, "failed to get applications")
		return nil, err
	}

	// Use a map to deduplicate apps (multiple servers can serve the same app)
	appMap := make(map[string]AppInfo)
	for _, server := range servers {
		app := server.GetApp()
		if app != nil {
			appMap[app.GetName()] = AppInfo{
				Name:       app.GetName(),
				PublicAddr: app.GetPublicAddr(),
				URI:        app.GetURI(),
				Labels:     app.GetAllLabels(),
			}
		}
	}

	result := make([]AppInfo, 0, len(appMap))
	for _, app := range appMap {
		result = append(result, app)
	}

	c.log.V(1).Info("fetched applications", "count", len(result))
	return result, nil
}

// GetClusterName returns the name of the connected Teleport cluster.
func (c *Client) GetClusterName(ctx context.Context) (string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	cn, err := c.client.GetClusterName(ctx)
	if err != nil {
		return "", err
	}
	return cn.GetClusterName(), nil
}
