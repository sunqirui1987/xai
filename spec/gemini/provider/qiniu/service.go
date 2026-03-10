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
	"net/url"
	"os"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/gemini"
)

const (
	Scheme          = "gemini-qiniu"
	DefaultBaseURL  = "https://api.qnaigc.com/v1/"
	OverseasBaseURL = "https://openai.sufy.com/v1/"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL       string
	httpClient    HTTPDoer
	clientOptions *clientOptions
}

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

func WithHTTPClient(cli HTTPDoer) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = cli
	}
}

// Service provides Gemini capabilities over Qiniu's OpenAI-compatible endpoints.
// It embeds gemini.Service to reuse all Gemini capabilities (Gen, GenStream, Operations,
// ImageFrom*, VideoFrom*, ReferenceImage, etc.) with a Qiniu-specific backend.
type Service struct {
	*gemini.Service
	client *client
}

// SetApiKey updates the API key at runtime. Implements xai.ApiKeySetter.
func (s *Service) SetApiKey(apiKey string) {
	s.client.setApiKey(apiKey)
}

// NewService creates a Qiniu Gemini service.
// The returned *Service supports SetApiKey(apiKey) for runtime API key updates.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("QINIU_API_KEY")
	}
	cfg := &clientConfig{baseURL: DefaultBaseURL}
	for _, opt := range opts {
		opt(cfg)
	}
	cli := newClient(apiKey, cfg.baseURL, cfg.httpClient, cfg.clientOptions)
	backend := newBackend(cli)
	return &Service{
		Service: gemini.NewWithBackend(backend),
		client:  cli,
	}
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
func Register(apiKey string, opts ...ClientOption) {
	cfg := &clientConfig{baseURL: DefaultBaseURL}
	for _, opt := range opts {
		opt(cfg)
	}
	defaultBaseURL := cfg.baseURL

	svc := NewService(apiKey, opts...)
	xai.Register(Scheme, func(_ context.Context, uri string) (xai.Service, error) {
		params, err := parseURIQuery(uri)
		if err != nil {
			return nil, err
		}
		key := apiKey
		if k := params.Get("key"); k != "" {
			key = k
		}
		if key == "" {
			key = os.Getenv("QINIU_API_KEY")
		}
		base := defaultBaseURL
		if b := params.Get("base"); b != "" {
			base = normalizeBaseURL(b)
		}
		if key == apiKey && base == defaultBaseURL {
			return svc, nil
		}
		var providerOpts []ClientOption
		providerOpts = append(providerOpts, WithBaseURL(base))
		if cfg.httpClient != nil {
			providerOpts = append(providerOpts, WithHTTPClient(cfg.httpClient))
		}
		return NewService(key, providerOpts...), nil
	})
}
