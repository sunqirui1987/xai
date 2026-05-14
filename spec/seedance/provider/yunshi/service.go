/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package yunshi

import (
	"context"
	"os"
	"time"

	"github.com/goplus/xai/spec/seedance"
)

// Service wraps seedance.Service and holds the Yunshi Cloud HTTP client for SetApiKey.
type Service struct {
	*seedance.Service
	client  *Client
	backend *backend
}

// SetApiKey updates the Yunshi Cloud API key on the underlying client.
func (s *Service) SetApiKey(apiKey string) {
	s.client.SetApiKey(apiKey)
}

// SeedanceService returns the embedded *seedance.Service for GenVideo Operation.Call.
func (s *Service) SeedanceService() *seedance.Service { return s.Service }

// NewService constructs a Seedance service with Yunshi Cloud backend.
func NewService(apiKey string, opts ...ClientOption) *Service {
	if apiKey == "" {
		apiKey = os.Getenv("YUNSHI_API_KEY")
	}
	client := NewClient(apiKey, opts...)
	return newServiceWithClient(client)
}

func newServiceWithClient(client *Client) *Service {
	b := newBackend(client)
	return &Service{
		Service: seedance.NewWithBackend(b),
		client:  client,
		backend: b,
	}
}

// Register registers seedance:// globally with a Yunshi-backed service.
func Register(apiKey string, opts ...ClientOption) {
	svc := NewService(apiKey, opts...)
	seedance.Register(svc)
}

// PushMaterial pushes a material URL to Volcengine through Yunshi Cloud.
func (s *Service) PushMaterial(ctx context.Context, req *PushMaterialRequest) (string, error) {
	return s.backend.PushMaterial(ctx, req)
}

// MaterialStatus returns the current Volcengine material status and optional error.
func (s *Service) MaterialStatus(ctx context.Context, assetID string) (string, string, error) {
	return s.backend.MaterialStatus(ctx, assetID)
}

// AwaitMaterialActive polls material status until it becomes Active.
func (s *Service) AwaitMaterialActive(ctx context.Context, assetID string, interval time.Duration, attempts int) error {
	return s.backend.AwaitMaterialActive(ctx, assetID, interval, attempts)
}
