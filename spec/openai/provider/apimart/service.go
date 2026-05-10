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

package apimart

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/openai"
)

const (
	// DefaultBaseURL is the default APIMart OpenAI-compatible base URL.
	DefaultBaseURL = "https://api.apimart.ai/v1/"

	// ModelGPTImage2 is the APIMart GPT-Image-2 model name.
	ModelGPTImage2 = "gpt-image-2"

	scheme = "apimart"

	DefaultBaseRetryDelay = 1 * time.Second
)

// ClientOption configures the APIMart service.
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL        string
	httpClient     *http.Client
	maxRetries     int
	baseRetryDelay time.Duration
	debugLog       bool
	logger         *log.Logger
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) ClientOption {
	return func(c *clientConfig) {
		c.baseURL = normalizeBaseURL(u)
	}
}

// WithHTTPClient overrides the HTTP client used for requests.
func WithHTTPClient(cl *http.Client) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = cl
	}
}

// WithRetry enables retry with exponential backoff on 429/5xx and transient network errors.
func WithRetry(maxRetries int, baseDelay time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.maxRetries = maxRetries
		if baseDelay > 0 {
			c.baseRetryDelay = baseDelay
		}
	}
}

// WithDebugLog enables extra response status/body logging. Curl logging is always enabled.
func WithDebugLog(enabled bool) ClientOption {
	return func(c *clientConfig) {
		c.debugLog = enabled
	}
}

// WithLogger sets the logger used for curl and debug lines. If nil, curl is printed to stdout.
func WithLogger(logger *log.Logger) ClientOption {
	return func(c *clientConfig) {
		c.logger = logger
	}
}

// Service wraps openai.Service builders with APIMart image operations.
type Service struct {
	*openai.Service
	baseURL        string
	apiKey         string
	httpClient     *http.Client
	maxRetries     int
	baseRetryDelay time.Duration
	debugLog       bool
	logger         *log.Logger
}

// SetApiKey updates the API key at runtime.
func (s *Service) SetApiKey(apiKey string) {
	s.apiKey = strings.TrimSpace(apiKey)
}

// Options returns APIMart options so image operations can honor runtime base URL overrides.
func (s *Service) Options() xai.OptionBuilder {
	return &options{}
}

// Actions returns the operations supported by the given model.
func (s *Service) Actions(model xai.Model) []xai.Action {
	if isGPTImageModel(model) {
		return []xai.Action{xai.GenImage, xai.EditImage}
	}
	return nil
}

// Operation creates an APIMart GPT-Image-2 operation.
func (s *Service) Operation(model xai.Model, action xai.Action) (xai.Operation, error) {
	if !isGPTImageModel(model) {
		return nil, xai.ErrNotFound
	}
	switch action {
	case xai.GenImage:
		return &genImage{model: normalizeModel(model)}, nil
	case xai.EditImage:
		return &editImage{model: normalizeModel(model)}, nil
	default:
		return nil, xai.ErrNotFound
	}
}

// GetTask resumes polling for an APIMart GPT-Image-2 task.
func (s *Service) GetTask(ctx context.Context, model xai.Model, action xai.Action, taskID string) (xai.OperationResponse, error) {
	if !isGPTImageModel(model) {
		return nil, xai.ErrNotFound
	}
	if action != xai.GenImage && action != xai.EditImage {
		return nil, xai.ErrNotFound
	}
	return s.getTask(ctx, s.baseURL, taskID)
}

// NewService creates an APIMart image service and panics on setup errors.
func NewService(apiKey string, opts ...ClientOption) *Service {
	svc, err := NewServiceWithError(apiKey, opts...)
	if err != nil {
		panic("apimart: " + err.Error())
	}
	return svc
}

// NewServiceWithError creates an APIMart image service.
func NewServiceWithError(apiKey string, opts ...ClientOption) (*Service, error) {
	if strings.TrimSpace(apiKey) == "" {
		apiKey = os.Getenv("APIMART_API_KEY")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("API key is required (set APIMART_API_KEY or pass apiKey)")
	}

	cfg := &clientConfig{
		baseURL:        DefaultBaseURL,
		maxRetries:     0,
		baseRetryDelay: DefaultBaseRetryDelay,
		debugLog:       true,
		logger:         log.Default(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	cfg.baseURL = normalizeBaseURL(cfg.baseURL)

	uri := openai.SchemeV1 + ":base=" + cfg.baseURL + "&key=" + url.QueryEscape(apiKey)
	baseSvc, err := openai.NewVideoOnly(context.Background(), uri)
	if err != nil {
		return nil, err
	}
	return &Service{
		Service:        baseSvc.(*openai.Service),
		baseURL:        cfg.baseURL,
		apiKey:         apiKey,
		httpClient:     cfg.httpClient,
		maxRetries:     cfg.maxRetries,
		baseRetryDelay: cfg.baseRetryDelay,
		debugLog:       cfg.debugLog,
		logger:         cfg.logger,
	}, nil
}

// Register registers the APIMart service with xai under scheme "apimart".
func Register(apiKey string, opts ...ClientOption) {
	svc, err := NewServiceWithError(apiKey, opts...)
	if err != nil {
		panic("apimart: " + err.Error())
	}
	xai.Register(scheme, func(ctx context.Context, uri string) (xai.Service, error) {
		query := strings.TrimPrefix(uri, scheme+":")
		if query == "" {
			return svc, nil
		}
		params, err := url.ParseQuery(query)
		if err != nil {
			return nil, err
		}
		key := apiKey
		if k := params.Get("key"); k != "" {
			key = k
		}
		if key == "" {
			key = os.Getenv("APIMART_API_KEY")
		}
		base := DefaultBaseURL
		if b := params.Get("base"); b != "" {
			base = normalizeBaseURL(b)
		}
		clientOpts := []ClientOption{}
		if base != DefaultBaseURL {
			clientOpts = append(clientOpts, WithBaseURL(base))
		}
		return NewServiceWithError(key, clientOpts...)
	})
}

type options struct {
	BaseURL string
}

func (p *options) WithBaseURL(base string) xai.OptionBuilder {
	p.BaseURL = normalizeBaseURL(base)
	return p
}

func (s *Service) operationBaseURL(opts xai.OptionBuilder) string {
	if override, ok := opts.(*options); ok && strings.TrimSpace(override.BaseURL) != "" {
		return normalizeBaseURL(override.BaseURL)
	}
	return normalizeBaseURL(s.baseURL)
}

func normalizeBaseURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = DefaultBaseURL
	}
	return strings.TrimSuffix(base, "/") + "/"
}

func (s *Service) logCurl(curlCmd string) {
	if s.logger != nil {
		s.logger.Printf("[apimart] curl command:\n%s", curlCmd)
		return
	}
	fmt.Println(curlCmd)
}

func (s *Service) logDebug(format string, args ...any) {
	if s.debugLog && s.logger != nil {
		s.logger.Printf("[apimart] "+format, args...)
	}
}
