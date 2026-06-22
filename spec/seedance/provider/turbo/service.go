/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package turbo

import (
	"os"

	"github.com/goplus/xai/spec/seedance"
)

// Service wraps seedance.Service and holds the Turbo HTTP client for SetApiKey.
type Service struct {
	*seedance.Service
	client  *Client
	backend *backend
}

// SetApiKey updates the Turbo API key on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
	if s.backend != nil && s.backend.assetClient != nil {
		s.backend.assetClient.SetApiKey(apiKey)
	}
}

// SeedanceService returns the embedded *seedance.Service for GenVideo Operation.Call.
func (s *Service) SeedanceService() *seedance.Service { return s.Service }

// NewService constructs a Seedance service with Turbo backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("TURBO_API_KEY")
	}
	client := NewClient(apiKey, opts...)
	backend := newBackend(client)
	return &Service{
		Service: seedance.NewWithBackend(backend),
		client:  client,
		backend: backend,
	}
}

// Register registers seedance:// globally with a Turbo-backed service.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	seedance.Register(svc)
}
