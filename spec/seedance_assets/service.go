/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package seedanceassets

import (
	"context"
	"strings"
	"time"
)

// Service provides Seedance 2.0 digital asset operations.
type Service struct {
	backend Backend
}

// NewWithBackend creates a Service with the given backend.
func NewWithBackend(backend Backend) *Service {
	if backend == nil {
		panic("seedance_assets: nil backend")
	}
	return &Service{backend: backend}
}

// NewService is an alias for NewWithBackend.
func NewService(backend Backend) *Service {
	return NewWithBackend(backend)
}

// CreateGroup creates one asset group in the provider-managed digital asset library.
func (s *Service) CreateGroup(ctx context.Context, req *CreateAssetGroupRequest) (*AssetGroup, error) {
	return s.backend.CreateGroup(ctx, req)
}

// UploadAsset uploads one asset into the provider-managed digital asset library.
func (s *Service) UploadAsset(ctx context.Context, req *UploadAssetRequest) (*AssetUploadResult, error) {
	return s.backend.UploadAsset(ctx, req)
}

// GetAsset loads one asset from the provider-managed digital asset library.
func (s *Service) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	return s.backend.GetAsset(ctx, assetID)
}

// AwaitAsset polls GetAsset until the asset leaves Processing state or the context ends.
func (s *Service) AwaitAsset(ctx context.Context, assetID string, interval time.Duration) (*Asset, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	for {
		asset, err := s.GetAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		if !isAssetProcessing(asset.Status) {
			return asset, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func isAssetProcessing(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), AssetStatusProcessing)
}
