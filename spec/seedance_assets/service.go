/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package seedanceassets

import "context"

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

// UploadAsset uploads one asset into the provider-managed digital asset library.
func (s *Service) UploadAsset(ctx context.Context, req *UploadAssetRequest) (*AssetUploadResult, error) {
	return s.backend.UploadAsset(ctx, req)
}

// GetAsset loads one asset from the provider-managed digital asset library.
func (s *Service) GetAsset(ctx context.Context, assetID string) (*Asset, error) {
	return s.backend.GetAsset(ctx, assetID)
}
