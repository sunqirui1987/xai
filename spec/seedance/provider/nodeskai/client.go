/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package nodeskai implements seedance.Backend against the NoDesk AI video API.
package nodeskai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL         = "https://llm-gateway-api.nodesk.tech"
	DefaultPlatformBaseURL = "https://platform-api.nodesk.tech"
	pathOAuthToken         = "/api/v1/oauth/token"

	defaultHTTPTimeout = 120 * time.Second

	DefaultBaseRetryDelay = 1 * time.Second

	maxDebugBodyLen = 4096
)

// Client is the HTTP client for the NoDesk AI Seedance 2.0 API.
type Client struct {
	httpClient      *http.Client
	baseURL         string
	platformBaseURL string
	externalUserID  string
	apiKeyMu        sync.RWMutex
	apiKey          string
	clientID        string
	clientSecret    string
	tokenExpiresAt  time.Time
	maxRetries      int
	baseRetryDelay  time.Duration
	debugLog        bool
	logger          *log.Logger
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets the API base URL (without trailing slash).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		s := strings.TrimSuffix(strings.TrimSpace(url), "/")
		if s != "" {
			c.baseURL = s
		}
	}
}

// WithPlatformBaseURL sets the digital-assets API base URL (without trailing slash).
func WithPlatformBaseURL(url string) ClientOption {
	return func(c *Client) {
		s := strings.TrimSuffix(strings.TrimSpace(url), "/")
		if s != "" {
			c.platformBaseURL = s
		}
	}
}

// WithExternalUserID sets the X-External-User-Id header for platform digital-assets APIs.
func WithExternalUserID(userID string) ClientOption {
	return func(c *Client) {
		c.externalUserID = strings.TrimSpace(userID)
	}
}

// WithOAuthClientCredentials sets the OAuth2 client_credentials used to obtain access tokens.
func WithOAuthClientCredentials(clientID, clientSecret string) ClientOption {
	return func(c *Client) {
		c.clientID = strings.TrimSpace(clientID)
		c.clientSecret = strings.TrimSpace(clientSecret)
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithRetry enables retry with exponential backoff for retryable HTTP status codes
// (429, 5xx) and transient network failures.
func WithRetry(maxRetries int, baseDelay time.Duration) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
		if baseDelay > 0 {
			c.baseRetryDelay = baseDelay
		}
	}
}

// WithDebugLog enables logging of HTTP response status and body.
func WithDebugLog(enabled bool) ClientOption {
	return func(c *Client) {
		c.debugLog = enabled
	}
}

// WithLogger sets the logger for curl and debug lines. If nil, curl is printed to stdout.
func WithLogger(logger *log.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a NoDesk AI client. apiKey is an optional bearer access token.
// When empty, the client will try NODESKAI_ACCESS_TOKEN first, then OAuth client credentials.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("NODESKAI_ACCESS_TOKEN")
	}
	if apiKey == "" {
		apiKey = os.Getenv("NODESK_ACCESS_TOKEN")
	}
	c := &Client{
		httpClient:      &http.Client{Timeout: defaultHTTPTimeout},
		baseURL:         DefaultBaseURL,
		platformBaseURL: DefaultPlatformBaseURL,
		externalUserID:  strings.TrimSpace(firstNonEmptyEnv("NODESKAI_EXTERNAL_USER_ID", "NODESK_EXTERNAL_USER_ID")),
		apiKey:          apiKey,
		maxRetries:      0,
		baseRetryDelay:  DefaultBaseRetryDelay,
		debugLog:        true,
		logger:          log.Default(),
	}
	c.clientID = strings.TrimSpace(firstNonEmptyEnv("NODESKAI_CLIENT_ID", "NODESK_CLIENT_ID"))
	c.clientSecret = strings.TrimSpace(firstNonEmptyEnv("NODESKAI_CLIENT_SECRET", "NODESK_CLIENT_SECRET"))
	for _, o := range opts {
		o(c)
	}
	return c
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

// ApiKey returns the current bearer token.
func (c *Client) ApiKey() string {
	c.apiKeyMu.RLock()
	defer c.apiKeyMu.RUnlock()
	return c.apiKey
}

// SetApiKey updates the bearer token at runtime.
func (c *Client) SetApiKey(key string) {
	c.apiKeyMu.Lock()
	c.apiKey = strings.TrimSpace(key)
	c.tokenExpiresAt = time.Time{}
	c.apiKeyMu.Unlock()
}

// BaseURL returns the configured API base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// PlatformBaseURL returns the configured digital-assets API base URL.
func (c *Client) PlatformBaseURL() string {
	return c.platformBaseURL
}

// ExternalUserID returns the configured X-External-User-Id value for platform APIs.
func (c *Client) ExternalUserID() string {
	return c.externalUserID
}

// LogDebug writes a debug line when WithDebugLog(true) and logger is set.
func (c *Client) LogDebug(format string, args ...any) {
	if c.debugLog && c.logger != nil {
		c.logger.Printf("[nodeskai] "+format, args...)
	}
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
}

func (c *Client) buildCurlCommand(method, fullURL string, body []byte) string {
	apiKey := c.ApiKey()
	var cmd bytes.Buffer
	cmd.WriteString("curl -X ")
	cmd.WriteString(method)
	if apiKey != "" {
		cmd.WriteString(fmt.Sprintf(" -H 'Authorization: Bearer %s'", apiKey))
	}
	cmd.WriteString(" -H 'Content-Type: application/json'")
	if len(body) > 0 {
		cmd.WriteString(fmt.Sprintf(" -d '%s'", string(body)))
	}
	cmd.WriteString(fmt.Sprintf(" '%s'", fullURL))
	return cmd.String()
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (c *Client) ensureAccessToken(ctx context.Context) (string, error) {
	if token := c.ApiKey(); token != "" && !c.shouldRefreshToken() {
		return token, nil
	}

	c.apiKeyMu.Lock()
	defer c.apiKeyMu.Unlock()

	if c.apiKey != "" && !c.shouldRefreshTokenLocked() {
		return c.apiKey, nil
	}
	if c.clientID == "" || c.clientSecret == "" {
		if c.apiKey != "" {
			return c.apiKey, nil
		}
		return "", fmt.Errorf("nodeskai: missing access token or oauth credentials (set NODESKAI_ACCESS_TOKEN or NODESKAI_CLIENT_ID/NODESKAI_CLIENT_SECRET)")
	}

	token, expiresIn, err := c.requestAccessToken(ctx)
	if err != nil {
		return "", err
	}
	c.apiKey = token
	if expiresIn > 0 {
		c.tokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	} else {
		c.tokenExpiresAt = time.Now().Add(55 * time.Minute)
	}
	return c.apiKey, nil
}

func (c *Client) shouldRefreshToken() bool {
	c.apiKeyMu.RLock()
	defer c.apiKeyMu.RUnlock()
	return c.shouldRefreshTokenLocked()
}

func (c *Client) shouldRefreshTokenLocked() bool {
	if c.apiKey == "" {
		return true
	}
	if c.tokenExpiresAt.IsZero() {
		return false
	}
	refreshAt := c.tokenExpiresAt.Add(-5 * time.Minute)
	if refreshAt.Before(time.Now()) {
		refreshAt = c.tokenExpiresAt.Add(-30 * time.Second)
	}
	return time.Now().After(refreshAt)
}

func (c *Client) requestAccessToken(ctx context.Context) (token string, expiresIn int64, err error) {
	fullURL := c.platformBaseURL + pathOAuthToken
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	oauthCurl := fmt.Sprintf(
		"curl -X POST %s -H %s -d %s",
		shellQuote(fullURL),
		shellQuote("Content-Type: application/x-www-form-urlencoded"),
		shellQuote(form.Encode()),
	)
	c.LogDebug("oauth curl command:\n%s", oauthCurl)
	c.LogDebug("oauth request url=%s", fullURL)
	c.LogDebug("oauth request content_type=application/x-www-form-urlencoded")
	c.LogDebug("oauth request form=%q", form.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("nodeskai: create oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("nodeskai: oauth request failed: %w", err)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", 0, fmt.Errorf("nodeskai: read oauth response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("nodeskai: oauth HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var v struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &v); err != nil {
		return "", 0, fmt.Errorf("nodeskai: parse oauth response: %w", err)
	}
	if strings.TrimSpace(v.AccessToken) == "" {
		return "", 0, fmt.Errorf("nodeskai: oauth response missing access_token")
	}
	c.LogDebug("oauth token fetched expires_in=%d", v.ExpiresIn)
	return strings.TrimSpace(v.AccessToken), v.ExpiresIn, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	return c.doRequestToBase(ctx, method, c.baseURL, path, body)
}

func (c *Client) doRequestToBase(ctx context.Context, method, baseURL, path string, body any) ([]byte, error) {
	token, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	fullURL := baseURL + path

	var reqBodyBytes []byte
	if body != nil {
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("nodeskai: marshal request body: %w", err)
		}
	}

	curlCmd := c.buildCurlCommand(method, fullURL, reqBodyBytes)
	if c.logger != nil {
		c.logger.Printf("[nodeskai] curl command:\n%s", curlCmd)
	} else {
		fmt.Println(curlCmd)
	}

	if os.Getenv("NODESKAI_MOCK_CURL") != "" {
		c.LogDebug("NODESKAI_MOCK_CURL set, aborting before HTTP")
		return nil, errors.New("nodeskai: mock curl mode")
	}

	var lastErr error
	maxAttempts := c.maxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			c.LogDebug("retry %d/%d after %v", attempt, c.maxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		var reqBody io.Reader
		if len(reqBodyBytes) > 0 {
			reqBody = bytes.NewReader(reqBodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			return nil, fmt.Errorf("nodeskai: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		if baseURL == c.platformBaseURL && c.externalUserID != "" {
			req.Header.Set("X-External-User-Id", c.externalUserID)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("nodeskai: request failed: %w", err)
			c.LogDebug("network error (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("nodeskai: read response: %w", err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		reqID := resp.Header.Get("X-Request-Id")
		c.LogDebug("response status=%d request_id=%s", resp.StatusCode, reqID)
		if c.debugLog && len(respBody) > 0 {
			bodyText := string(respBody)
			if len(bodyText) > maxDebugBodyLen {
				bodyText = bodyText[:maxDebugBodyLen] + "..."
			}
			c.LogDebug("response body=%s", bodyText)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		lastErr = fmt.Errorf("nodeskai: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		if isRetryableStatus(resp.StatusCode) && attempt < c.maxRetries {
			continue
		}
		return nil, lastErr
	}

	return nil, lastErr
}

// PostJSON performs a JSON POST request.
func (c *Client) PostJSON(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// GetJSON performs a GET request.
func (c *Client) GetJSON(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// PostPlatformJSON performs a JSON POST request against the platform API base URL.
func (c *Client) PostPlatformJSON(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequestToBase(ctx, http.MethodPost, c.platformBaseURL, path, body)
}

// PostMultipart performs a multipart POST request against the platform API base URL.
func (c *Client) PostMultipart(ctx context.Context, path string, contentType string, body []byte) ([]byte, error) {
	return c.doRawRequestToBase(ctx, http.MethodPost, c.platformBaseURL, path, contentType, body)
}

func (c *Client) doRawRequestToBase(ctx context.Context, method, baseURL, path, contentType string, body []byte) ([]byte, error) {
	token, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	fullURL := baseURL + path

	curlCmd := c.buildCurlCommand(method, fullURL, nil)
	if c.logger != nil {
		c.logger.Printf("[nodeskai] curl command:\n%s", curlCmd)
	} else {
		fmt.Println(curlCmd)
	}

	if os.Getenv("NODESKAI_MOCK_CURL") != "" {
		c.LogDebug("NODESKAI_MOCK_CURL set, aborting before HTTP")
		return nil, errors.New("nodeskai: mock curl mode")
	}

	var lastErr error
	maxAttempts := c.maxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			c.LogDebug("retry %d/%d after %v", attempt, c.maxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("nodeskai: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if baseURL == c.platformBaseURL && c.externalUserID != "" {
			req.Header.Set("X-External-User-Id", c.externalUserID)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("nodeskai: request failed: %w", err)
			c.LogDebug("network error (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("nodeskai: read response: %w", err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		reqID := resp.Header.Get("X-Request-Id")
		c.LogDebug("response status=%d request_id=%s", resp.StatusCode, reqID)
		if c.debugLog && len(respBody) > 0 {
			bodyText := string(respBody)
			if len(bodyText) > maxDebugBodyLen {
				bodyText = bodyText[:maxDebugBodyLen] + "..."
			}
			c.LogDebug("response body=%s", bodyText)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		lastErr = fmt.Errorf("nodeskai: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		if isRetryableStatus(resp.StatusCode) && attempt < c.maxRetries {
			continue
		}
		return nil, lastErr
	}

	return nil, lastErr
}
