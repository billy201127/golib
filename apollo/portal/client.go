package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Constants definition
const (
	DefaultTimeout = 30 * time.Second
	APIPathFormat  = "/openapi/v1/envs/%s/apps/%s/clusters/%s/namespaces/%s"
)

// PortalClient Apollo configuration management client
type PortalClient struct {
	PortalURL  string
	Token      string
	AppID      string
	Env        string
	Cluster    string
	Namespace  string
	Operator   string
	HTTPClient *http.Client
}

// NewPortalClient creates a new Portal client instance
func NewPortalClient(config ApolloConfig) *PortalClient {
	client := &PortalClient{
		PortalURL: config.PortalURL,
		Token:     config.Token,
		AppID:     config.AppID,
		Env:       config.Env,
		Cluster:   config.Cluster,
		Namespace: config.Namespace,
		Operator:  config.Operator,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	// Set default values
	if client.Cluster == "" {
		client.Cluster = "default"
	}
	if client.Namespace == "" {
		client.Namespace = "application"
	}
	if client.Operator == "" {
		client.Operator = "apollo"
	}

	return client
}

// ApolloConfig Apollo client configuration
type ApolloConfig struct {
	PortalURL string
	Token     string
	AppID     string
	Env       string
	Cluster   string
	Namespace string
	Operator  string
}

// Item configuration item structure
type Item struct {
	Key                        string `json:"key"`
	Value                      string `json:"value"`
	Comment                    string `json:"comment,omitempty"`
	DataChangeCreatedBy        string `json:"dataChangeCreatedBy,omitempty"`
	DataChangeLastModifiedBy   string `json:"dataChangeLastModifiedBy,omitempty"`
	DataChangeCreatedTime      string `json:"dataChangeCreatedTime,omitempty"`
	DataChangeLastModifiedTime string `json:"dataChangeLastModifiedTime,omitempty"`
}

// Release release information structure
type Release struct {
	ReleaseTitle   string `json:"releaseTitle"`
	ReleaseComment string `json:"releaseComment"`
	ReleasedBy     string `json:"releasedBy"`
}

// APIResponse common API response structure
type APIResponse struct {
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// CreateItem creates a new configuration item
func (c *PortalClient) CreateItem(ctx context.Context, key, value, comment string) error {
	if key == "" {
		return fmt.Errorf("configuration item key cannot be empty")
	}

	url := c.buildItemURL("")
	item := Item{
		Key:                 key,
		Value:               value,
		Comment:             comment,
		DataChangeCreatedBy: c.Operator,
	}

	return c.doRequest(ctx, http.MethodPost, url, item)
}

// UpdateItem updates an existing configuration item
func (c *PortalClient) UpdateItem(ctx context.Context, key, value, comment string) error {
	if key == "" {
		return fmt.Errorf("configuration item key cannot be empty")
	}

	url := c.buildItemURL(key)
	item := Item{
		Key:                      key,
		Value:                    value,
		Comment:                  comment,
		DataChangeLastModifiedBy: c.Operator,
	}

	return c.doRequest(ctx, http.MethodPut, url, item)
}

// DeleteItem deletes a configuration item
func (c *PortalClient) DeleteItem(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("configuration item key cannot be empty")
	}

	url := c.buildItemURL(key) + "?operator=" + url.QueryEscape(c.Operator)
	return c.doRequest(ctx, http.MethodDelete, url, nil)
}

// GetItem retrieves a configuration item
func (c *PortalClient) GetItem(ctx context.Context, key string) (*Item, error) {
	if key == "" {
		return nil, fmt.Errorf("configuration item key cannot be empty")
	}

	url := c.buildItemURL(key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var item Item
		if err := json.Unmarshal(body, &item); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return &item, nil
	}

	return nil, fmt.Errorf("failed to get configuration item: %s (status=%d)", string(body), resp.StatusCode)
}

// ListItems retrieves all configuration items in the namespace
func (c *PortalClient) ListItems(ctx context.Context) ([]Item, error) {
	url := c.buildNamespaceURL() + "/items"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var items []Item
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return items, nil
	}

	return nil, fmt.Errorf("failed to get configuration item list: %s (status=%d)", string(body), resp.StatusCode)
}

// PublishConfig publishes configuration
func (c *PortalClient) PublishConfig(ctx context.Context, title, comment string) error {
	if title == "" {
		return fmt.Errorf("release title cannot be empty")
	}

	url := c.buildNamespaceURL() + "/releases"
	release := Release{
		ReleaseTitle:   title,
		ReleaseComment: comment,
		ReleasedBy:     c.Operator,
	}

	return c.doRequest(ctx, http.MethodPost, url, release)
}

// buildNamespaceURL builds the namespace base URL
func (c *PortalClient) buildNamespaceURL() string {
	return fmt.Sprintf("%s"+APIPathFormat,
		c.PortalURL, c.Env, c.AppID, c.Cluster, c.Namespace)
}

// buildItemURL builds the configuration item URL
func (c *PortalClient) buildItemURL(key string) string {
	baseURL := c.buildNamespaceURL() + "/items"
	if key != "" {
		return baseURL + "/" + url.PathEscape(key)
	}
	return baseURL
}

// setHeaders sets request headers
func (c *PortalClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)
	req.Header.Set("User-Agent", "Apollo-Go-Client/1.0")
}

// doRequest executes HTTP request - common method
func (c *PortalClient) doRequest(ctx context.Context, method, url string, payload interface{}) error {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to serialize request data: %w", err)
		}
		bodyReader = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ Operation successful: %s %s\n", method, url)
		if len(respBody) > 0 && len(respBody) < 1000 { // Avoid printing overly long responses
			fmt.Printf("   Response: %s\n", string(respBody))
		}
		return nil
	}

	return fmt.Errorf("request failed: %s (status=%d, method=%s, url=%s)",
		string(respBody), resp.StatusCode, method, url)
}
