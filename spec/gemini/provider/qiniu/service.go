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
	"context"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/gemini"
)

const (
	// Scheme is the xai URI scheme used by this provider.
	Scheme = "gemini-qiniu"

	// DefaultBaseURL is the default Qiniu API base URL (domestic).
	DefaultBaseURL = "https://api.qnaigc.com/v1/"
	// OverseasBaseURL is the overseas API base URL.
	OverseasBaseURL = "https://openai.sufy.com/v1/"
)

// ClientOption configures the Qiniu Gemini provider.
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL        string
	httpClient     HTTPDoer
	maxRetries     int
	baseRetryDelay time.Duration
	debugLog       bool
	logger         *log.Logger
}

// WithBaseURL sets the base URL for API requests.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *clientConfig) {
		c.baseURL = normalizeBaseURL(baseURL)
	}
}

func normalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return DefaultBaseURL
	}
	return strings.TrimSuffix(baseURL, "/") + "/"
}

// WithHTTPClient sets a custom HTTP client for provider HTTP calls.
func WithHTTPClient(cli HTTPDoer) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = cli
	}
}

// WithRetry enables retry with exponential backoff for retryable errors.
// maxRetries is the maximum number of retry attempts (0 means no retry).
// baseDelay is the initial delay between retries (doubles with each attempt).
func WithRetry(maxRetries int, baseDelay time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.maxRetries = maxRetries
		c.baseRetryDelay = baseDelay
	}
}

// WithDebugLog enables debug logging for HTTP requests and responses.
func WithDebugLog(enabled bool) ClientOption {
	return func(c *clientConfig) {
		c.debugLog = enabled
	}
}

// WithLogger sets a custom logger for debug output.
func WithLogger(logger *log.Logger) ClientOption {
	return func(c *clientConfig) {
		c.logger = logger
	}
}

// NewBackend creates a Qiniu backend implementation for spec/gemini.
func NewBackend(token string, opts ...ClientOption) gemini.Backend {
	if token == "" {
		token = os.Getenv("QINIU_API_KEY")
	}
	cfg := &clientConfig{
		baseURL:        DefaultBaseURL,
		debugLog:       true,
		logger:         log.Default(),
		baseRetryDelay: DefaultBaseRetryDelay,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	clientOpts := &clientOptions{
		maxRetries:     cfg.maxRetries,
		baseRetryDelay: cfg.baseRetryDelay,
		debugLog:       cfg.debugLog,
		logger:         cfg.logger,
	}
	return newBackend(newClient(token, cfg.baseURL, cfg.httpClient, clientOpts))
}

// NewService creates a spec/gemini Service using Qiniu backend.
func NewService(token string, opts ...ClientOption) *gemini.Service {
	return gemini.NewWithBackend(NewBackend(token, opts...))
}

func parseURIQuery(uri string) (url.Values, error) {
	if strings.HasPrefix(uri, Scheme+":") {
		uri = strings.TrimPrefix(uri, Scheme+":")
	}
	if uri == "" {
		return url.Values{}, nil
	}
	return url.ParseQuery(uri)
}

// Register registers the Qiniu-backed Gemini service with xai under scheme "gemini-qiniu".
//
// After calling Register(token), xai.New(ctx, "gemini-qiniu:") returns the provider service.
// URI overrides are supported:
//   - gemini-qiniu:key=xxx
//   - gemini-qiniu:base=https://openai.sufy.com/v1/&key=xxx
//
// Token can be empty to use QINIU_API_KEY from environment.
func Register(token string, opts ...ClientOption) {
	cfg := &clientConfig{
		baseURL:        DefaultBaseURL,
		debugLog:       true,
		logger:         log.Default(),
		baseRetryDelay: DefaultBaseRetryDelay,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	defaultBaseURL := cfg.baseURL

	svc := NewService(token, opts...)
	xai.Register(Scheme, func(_ context.Context, uri string) (xai.Service, error) {
		params, err := parseURIQuery(uri)
		if err != nil {
			return nil, err
		}
		tok := token
		if key := params.Get("key"); key != "" {
			tok = key
		}
		if tok == "" {
			tok = os.Getenv("QINIU_API_KEY")
		}
		base := defaultBaseURL
		if b := params.Get("base"); b != "" {
			base = normalizeBaseURL(b)
		}
		if tok == token && base == defaultBaseURL {
			return svc, nil
		}
		var providerOpts []ClientOption
		providerOpts = append(providerOpts, WithBaseURL(base))
		if cfg.httpClient != nil {
			providerOpts = append(providerOpts, WithHTTPClient(cfg.httpClient))
		}
		if cfg.maxRetries > 0 {
			providerOpts = append(providerOpts, WithRetry(cfg.maxRetries, cfg.baseRetryDelay))
		}
		if cfg.logger != nil {
			providerOpts = append(providerOpts, WithLogger(cfg.logger))
		}
		providerOpts = append(providerOpts, WithDebugLog(cfg.debugLog))
		return NewService(tok, providerOpts...), nil
	})
}
