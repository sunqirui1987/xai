/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package seedanceassets

import (
	"fmt"
	"io"
)

const (
	AssetTypeImage = "Image"
	AssetTypeVideo = "Video"
	AssetTypeAudio = "Audio"

	AssetStatusProcessing = "Processing"
	AssetStatusActive     = "Active"
	AssetStatusFailed     = "Failed"
)

// UploadAssetRequest describes a single digital asset upload request.
type UploadAssetRequest struct {
	GroupID  string
	Name     string
	FileName string
	File     io.Reader
}

func (r *UploadAssetRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("seedance_assets: nil upload asset request")
	}
	if r.GroupID == "" {
		return fmt.Errorf("seedance_assets: group_id is required")
	}
	if r.File == nil {
		return fmt.Errorf("seedance_assets: file is required")
	}
	if r.FileName == "" {
		return fmt.Errorf("seedance_assets: file_name is required")
	}
	return nil
}

// CreateAssetGroupRequest describes one asset-group creation request.
type CreateAssetGroupRequest struct {
	Name        string
	Description string
}

func (r *CreateAssetGroupRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("seedance_assets: nil create asset group request")
	}
	if r.Name == "" {
		return fmt.Errorf("seedance_assets: group name is required")
	}
	return nil
}

// ListAssetGroupsRequest describes one asset-group listing request.
type ListAssetGroupsRequest struct {
	Name       string
	PageNumber int
	PageSize   int
}

func (r *ListAssetGroupsRequest) Normalize() *ListAssetGroupsRequest {
	if r == nil {
		return &ListAssetGroupsRequest{PageNumber: 1, PageSize: 20}
	}
	out := *r
	if out.PageNumber <= 0 {
		out.PageNumber = 1
	}
	if out.PageSize <= 0 {
		out.PageSize = 20
	}
	return &out
}

// AssetUploadResult is returned after an upload request is accepted.
type AssetUploadResult struct {
	Success   bool
	AssetID   string
	AssetType string
	Status    string
	TOSURL    string
	FileName  string
	FileSize  int64
}

// Asset is the normalized asset detail returned by provider asset libraries.
type Asset struct {
	ID         string
	GroupID    string
	Name       string
	Type       string
	Status     string
	URL        string
	CreateTime string
}

// AssetGroup is one provider-managed asset group.
type AssetGroup struct {
	ID          string
	Name        string
	Description string
	Status      string
	AssetCount  int
	CreateTime  string
}

// AssetGroupList is one paged asset-group list result.
type AssetGroupList struct {
	Items      []*AssetGroup
	TotalCount int
	PageNumber int
	PageSize   int
}
