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
	"sync"

	"github.com/go-logr/logr"
	"github.com/gravitational/teleport/api/client"
)

// Config holds the configuration for the Teleport client.
type Config struct {
	// ProxyAddr is the address of the Teleport proxy or auth server.
	ProxyAddr string
	// IdentityFile is the path to the identity file for authentication.
	IdentityFile string
	// Insecure skips TLS certificate verification.
	Insecure bool
	// Log is the logger to use.
	Log logr.Logger
}

// Client wraps the Teleport API client.
type Client struct {
	client    *client.Client
	log       logr.Logger
	connected bool
	mu        sync.RWMutex
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

	creds := client.LoadIdentityFile(cfg.IdentityFile)

	c, err := client.New(context.Background(), client.Config{
		Addrs:                    []string{cfg.ProxyAddr},
		Credentials:              []client.Credentials{creds},
		InsecureAddressDiscovery: cfg.Insecure,
	})
	if err != nil {
		return nil, err
	}

	cfg.Log.Info("connected to Teleport successfully")

	return &Client{
		client:    c,
		log:       cfg.Log,
		connected: true,
	}, nil
}

// Close closes the Teleport client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return c.client.Close()
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetNodes returns all nodes registered in Teleport.
func (c *Client) GetNodes(ctx context.Context) ([]NodeInfo, error) {
	c.log.V(1).Info("fetching nodes from Teleport")

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
	cn, err := c.client.GetClusterName(ctx)
	if err != nil {
		return "", err
	}
	return cn.GetClusterName(), nil
}
