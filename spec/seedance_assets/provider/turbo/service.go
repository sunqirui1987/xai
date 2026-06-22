/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package turbo

import (
	"os"

	seedanceturbo "github.com/goplus/xai/spec/seedance/provider/turbo"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const DefaultBaseURL = "https://inference.turbo-api.com"

type ClientOption = seedanceturbo.ClientOption
type Client = seedanceturbo.Client

var (
	WithBaseURL    = seedanceturbo.WithBaseURL
	WithHTTPClient = seedanceturbo.WithHTTPClient
	WithRetry      = seedanceturbo.WithRetry
	WithDebugLog   = seedanceturbo.WithDebugLog
	WithLogger     = seedanceturbo.WithLogger
)

// Service wraps seedanceassets.Service and holds the Turbo HTTP client for SetApiKey.
type Service struct {
	*seedanceassets.Service
	client *Client
}

// SetApiKey updates the Turbo API key on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// NewService constructs a Seedance assets service with Turbo backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("TURBO_API_KEY")
	}
	opts = append([]ClientOption{WithBaseURL(DefaultBaseURL)}, opts...)
	client := seedanceturbo.NewClient(apiKey, opts...)
	return &Service{
		Service: seedanceassets.NewWithBackend(newBackend(client)),
		client:  client,
	}
}
