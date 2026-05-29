/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package qiniu

import (
	"os"

	seedanceqiniu "github.com/goplus/xai/spec/seedance/provider/qiniu"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	DefaultBaseURL  = "https://openai.qiniu.com"
	OverseasBaseURL = "https://openai.sufy.com"
)

type ClientOption = seedanceqiniu.ClientOption
type Client = seedanceqiniu.Client

var (
	WithBaseURL    = seedanceqiniu.WithBaseURL
	WithHTTPClient = seedanceqiniu.WithHTTPClient
	WithRetry      = seedanceqiniu.WithRetry
	WithDebugLog   = seedanceqiniu.WithDebugLog
	WithLogger     = seedanceqiniu.WithLogger
)

// Service wraps seedanceassets.Service and holds the Qiniu HTTP client for SetApiKey.
type Service struct {
	*seedanceassets.Service
	client *Client
}

// SetApiKey updates the Qiniu API key on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// NewService constructs a Seedance assets service with Qiniu backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("QINIU_API_KEY")
	}
	opts = append([]ClientOption{WithBaseURL(DefaultBaseURL)}, opts...)
	client := seedanceqiniu.NewClient(apiKey, opts...)
	return &Service{
		Service: seedanceassets.NewWithBackend(newBackend(client)),
		client:  client,
	}
}
