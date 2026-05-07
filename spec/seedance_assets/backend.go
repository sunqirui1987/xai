/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package seedanceassets

import "context"

// Backend is the pluggable transport for Seedance 2.0 digital asset services.
type Backend interface {
	CreateGroup(ctx context.Context, req *CreateAssetGroupRequest) (*AssetGroup, error)
	ListGroups(ctx context.Context, req *ListAssetGroupsRequest) (*AssetGroupList, error)
	UploadAsset(ctx context.Context, req *UploadAssetRequest) (*AssetUploadResult, error)
	GetAsset(ctx context.Context, assetID string) (*Asset, error)
}
