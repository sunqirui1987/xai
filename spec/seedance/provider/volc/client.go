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

// Package volc implements seedance.Backend against Volcengine Ark HTTP APIs.
//
// API reference:
//   - https://www.volcengine.com/docs/82379/1520757?lang=zh — POST create task
//   - https://www.volcengine.com/docs/82379/1521309?lang=zh — GET query task
//
// The HTTP Client mirrors spec/kling/provider/qiniu: by default it logs an equivalent
// curl command and optional response details (see WithDebugLog / WithLogger).
package volc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBaseURL is the Volc Ark Beijing endpoint for contents generation.
	DefaultBaseURL = "https://ark.cn-beijing.volces.com"

	defaultHTTPTimeout = 120 * time.Second

	// DefaultBaseRetryDelay is the initial backoff when WithRetry is used.
	DefaultBaseRetryDelay = 1 * time.Second

	// maxDebugBodyLen truncates logged response bodies to avoid huge logs.
	maxDebugBodyLen = 4096
)

// Client is the HTTP client for Volc Ark Seedance (contents generations) API.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	apiKeyMu       sync.RWMutex
	apiKey         string
	maxRetries     int
	baseRetryDelay time.Duration
	debugLog       bool
	logger         *log.Logger
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

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithRetry enables retry with exponential backoff for retryable HTTP status codes
// (429, 5xx) and transient network failures. maxRetries is the number of *retries*
// after the first attempt (0 means no retry, same default as Qiniu client).
func WithRetry(maxRetries int, baseDelay time.Duration) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
		if baseDelay > 0 {
			c.baseRetryDelay = baseDelay
		}
	}
}

// WithDebugLog enables logging of HTTP response status and body (curl command is always
// emitted when logger is non-nil, matching Qiniu behavior).
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

// NewClient creates an Ark API client. apiKey may be empty to use ARK_API_KEY from the environment.
// Defaults match Qiniu: debugLog=true, logger=log.Default(), maxRetries=0.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ARK_API_KEY")
	}
	c := &Client{
		httpClient:     &http.Client{Timeout: defaultHTTPTimeout},
		baseURL:        DefaultBaseURL,
		apiKey:         apiKey,
		maxRetries:     0,
		baseRetryDelay: DefaultBaseRetryDelay,
		debugLog:       true,
		logger:         log.Default(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ApiKey returns the current bearer token (API key).
func (c *Client) ApiKey() string {
	c.apiKeyMu.RLock()
	defer c.apiKeyMu.RUnlock()
	return c.apiKey
}

// SetApiKey updates the bearer token at runtime.
func (c *Client) SetApiKey(key string) {
	c.apiKeyMu.Lock()
	c.apiKey = key
	c.apiKeyMu.Unlock()
}

// BaseURL returns the configured API base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// LogDebug writes a debug line when WithDebugLog(true) and logger is set.
// Used by the volc backend for task lifecycle messages.
func (c *Client) LogDebug(format string, args ...any) {
	c.logDebug(format, args...)
}

func (c *Client) logDebug(format string, args ...any) {
	if c.debugLog && c.logger != nil {
		c.logger.Printf("[volc] "+format, args...)
	}
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == 429 || statusCode == 500 || statusCode == 502 || statusCode == 503 || statusCode == 504
}

// buildCurlCommand builds an equivalent curl command for debugging (includes full Bearer token; do not paste into public channels).
func (c *Client) buildCurlCommand(method, fullURL string, body []byte) string {
	apiKey := c.ApiKey()
	var cmd bytes.Buffer
	cmd.WriteString("curl -X ")
	cmd.WriteString(method)
	cmd.WriteString(fmt.Sprintf(" -H 'Authorization: Bearer %s'", apiKey))
	cmd.WriteString(" -H 'Content-Type: application/json'")
	if len(body) > 0 {
		cmd.WriteString(fmt.Sprintf(" -d '%s'", string(body)))
	}
	cmd.WriteString(fmt.Sprintf(" '%s'", fullURL))
	return cmd.String()
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	if strings.TrimSpace(c.ApiKey()) == "" {
		return nil, fmt.Errorf("volc: missing API key (set ARK_API_KEY or use SetApiKey)")
	}
	fullURL := c.baseURL + path

	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("volc: marshal request body: %w", err)
		}
	}

	curlCmd := c.buildCurlCommand(method, fullURL, reqBodyBytes)
	if c.logger != nil {
		c.logger.Printf("[volc] curl command:\n%s", curlCmd)
	} else {
		fmt.Println(curlCmd)
	}

	if os.Getenv("VOLC_MOCK_CURL") != "" {
		c.logDebug("VOLC_MOCK_CURL set, aborting before HTTP")
		return nil, errors.New("volc: mock curl mode")
	}

	var lastErr error
	maxAttempts := c.maxRetries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			c.logDebug("retry %d/%d after %v", attempt, c.maxRetries, delay)
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
			return nil, fmt.Errorf("volc: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey())
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("volc: request failed: %w", err)
			c.logDebug("network error (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("volc: read response: %w", err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		reqID := resp.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = resp.Header.Get("X-Request-ID")
		}
		c.logDebug("response status=%d request_id=%q", resp.StatusCode, reqID)
		if c.debugLog && len(respBody) > 0 {
			bodyStr := string(respBody)
			if len(bodyStr) > maxDebugBodyLen {
				bodyStr = bodyStr[:maxDebugBodyLen] + "...(truncated)"
			}
			c.logDebug("response body: %s", bodyStr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		apiErr := fmt.Errorf("volc: HTTP %d: %s", resp.StatusCode, truncateErr(string(respBody), 512))
		if isRetryableStatus(resp.StatusCode) && attempt < c.maxRetries {
			lastErr = apiErr
			c.logDebug("retryable status %d (attempt %d/%d)", resp.StatusCode, attempt+1, maxAttempts)
			continue
		}
		return nil, apiErr
	}

	return nil, lastErr
}

// PostJSON sends POST to path (must start with /) with a JSON-marshalable body.
func (c *Client) PostJSON(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// GetJSON sends GET to path (must start with /).
func (c *Client) GetJSON(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

func truncateErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
