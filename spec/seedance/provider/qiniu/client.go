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

package qiniu

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
	DefaultBaseURL = "https://api.qnaigc.com"

	defaultHTTPTimeout = 120 * time.Second

	DefaultBaseRetryDelay = 1 * time.Second

	maxDebugBodyLen = 4096
)

// Client is the HTTP client for the Qiniu/Qnagic Seedance contents generations API.
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

// NewClient creates a Qiniu Seedance API client. apiKey may be empty to use QINIU_API_KEY.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("QINIU_API_KEY")
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

// ApiKey returns the current bearer token.
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
func (c *Client) LogDebug(format string, args ...any) {
	if c.debugLog && c.logger != nil {
		c.logger.Printf("[qiniu-seedance] "+format, args...)
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
		return nil, fmt.Errorf("qiniu-seedance: missing API key (set QINIU_API_KEY or use SetApiKey)")
	}
	fullURL := c.baseURL + path

	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("qiniu-seedance: marshal request body: %w", err)
		}
	}

	curlCmd := c.buildCurlCommand(method, fullURL, reqBodyBytes)
	if c.logger != nil {
		c.logger.Printf("[qiniu-seedance] curl command:\n%s", curlCmd)
	} else {
		fmt.Println(curlCmd)
	}

	if os.Getenv("QINIU_MOCK_CURL") != "" {
		c.LogDebug("QINIU_MOCK_CURL set, aborting before HTTP")
		return nil, errors.New("qiniu-seedance: mock curl mode")
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
			return nil, fmt.Errorf("qiniu-seedance: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey())
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("qiniu-seedance: request failed: %w", err)
			c.LogDebug("network error (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			if attempt < c.maxRetries {
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("qiniu-seedance: read response: %w", err)
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

		lastErr = fmt.Errorf("qiniu-seedance: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		if isRetryableStatus(resp.StatusCode) && attempt < c.maxRetries {
			continue
		}
		return nil, lastErr
	}

	return nil, lastErr
}

// PostJSON sends a POST request with a JSON body and returns raw response bytes.
func (c *Client) PostJSON(ctx context.Context, path string, body any) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// GetJSON sends a GET request and returns raw response bytes.
func (c *Client) GetJSON(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}
