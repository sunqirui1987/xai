/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package qiniu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	seedanceqiniu "github.com/goplus/xai/spec/seedance/provider/qiniu"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	pathCreateGroup = "/v1/asset-groups"
	pathCreateAsset = "/v1/assets"
	pathGetAsset    = "/v1/assets/"

	defaultAssetGroupType = "aigc"
)

type backend struct {
	client *seedanceqiniu.Client
}

func newBackend(client *seedanceqiniu.Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedanceassets.Backend backed by the Qiniu OpenAI-compatible assets API.
func NewBackend(client *seedanceqiniu.Client) seedanceassets.Backend {
	return newBackend(client)
}

func (b *backend) CreateGroup(ctx context.Context, req *seedanceassets.CreateAssetGroupRequest) (*seedanceassets.AssetGroup, error) {
	body, err := buildCreateGroupBody(req)
	if err != nil {
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathCreateGroup, body)
	if err != nil {
		return nil, err
	}
	return parseCreateGroupResponse(raw)
}

func (b *backend) ListGroups(ctx context.Context, req *seedanceassets.ListAssetGroupsRequest) (*seedanceassets.AssetGroupList, error) {
	return nil, fmt.Errorf("seedance_assets/qiniu: ListGroups is not supported by Qiniu assets API")
}

func (b *backend) UploadAsset(ctx context.Context, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	body, err := buildCreateAssetBody(req)
	if err != nil {
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathCreateAsset, body)
	if err != nil {
		return nil, err
	}
	return parseCreateAssetResponse(raw)
}

func (b *backend) GetAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, fmt.Errorf("seedance_assets/qiniu: qassetid is required")
	}
	raw, err := b.client.GetJSON(ctx, pathGetAsset+url.PathEscape(assetID))
	if err != nil {
		return nil, err
	}
	return parseGetAssetResponse(raw)
}

func buildCreateGroupBody(req *seedanceassets.CreateAssetGroupRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("seedance_assets/qiniu: nil create asset group request")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return nil, fmt.Errorf("seedance_assets/qiniu: model is required")
	}
	groupType := strings.TrimSpace(req.Type)
	if groupType == "" {
		groupType = defaultAssetGroupType
	}
	body := map[string]any{
		"type":  groupType,
		"model": model,
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		body["name"] = name
	}
	if description := strings.TrimSpace(req.Description); description != "" {
		body["description"] = description
	}
	return body, nil
}

func buildCreateAssetBody(req *seedanceassets.UploadAssetRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("seedance_assets/qiniu: nil upload asset request")
	}
	assetType := normalizeAssetType(req.AssetType)
	if assetType == "" {
		return nil, fmt.Errorf("seedance_assets/qiniu: asset type is required")
	}
	assetURL := strings.TrimSpace(req.URL)
	if assetURL == "" {
		return nil, fmt.Errorf("seedance_assets/qiniu: url is required")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return nil, fmt.Errorf("seedance_assets/qiniu: model is required")
	}
	body := map[string]any{
		"type":  assetType,
		"url":   assetURL,
		"model": model,
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		body["name"] = name
	}
	if groupID := strings.TrimSpace(req.GroupID); groupID != "" {
		body["group_id"] = groupID
	}
	return body, nil
}

func normalizeAssetType(assetType string) string {
	switch strings.ToLower(strings.TrimSpace(assetType)) {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	default:
		return strings.TrimSpace(assetType)
	}
}

func parseCreateGroupResponse(raw []byte) (*seedanceassets.AssetGroup, error) {
	var v qiniuAssetGroup
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/qiniu: parse create group response: %w", err)
	}
	return v.toAssetGroup(), nil
}

func parseCreateAssetResponse(raw []byte) (*seedanceassets.AssetUploadResult, error) {
	var v qiniuAsset
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/qiniu: parse create asset response: %w", err)
	}
	return &seedanceassets.AssetUploadResult{
		Success:    !strings.EqualFold(strings.TrimSpace(v.Status), seedanceassets.AssetStatusFailed),
		AssetID:    strings.TrimSpace(v.ID),
		AssetType:  strings.TrimSpace(v.Type),
		Name:       strings.TrimSpace(v.Name),
		Model:      strings.TrimSpace(v.Model),
		Status:     strings.TrimSpace(v.Status),
		GroupID:    strings.TrimSpace(v.GroupID),
		FailReason: strings.TrimSpace(v.FailReason),
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}, nil
}

func parseGetAssetResponse(raw []byte) (*seedanceassets.Asset, error) {
	var v qiniuAsset
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("seedance_assets/qiniu: parse get asset response: %w", err)
	}
	return v.toAsset(), nil
}

type qiniuAssetGroup struct {
	ID          string `json:"qgroupid"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default"`
	FailReason  string `json:"fail_reason"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func (v qiniuAssetGroup) toAssetGroup() *seedanceassets.AssetGroup {
	return &seedanceassets.AssetGroup{
		ID:          strings.TrimSpace(v.ID),
		Type:        strings.TrimSpace(v.Type),
		Name:        strings.TrimSpace(v.Name),
		Description: strings.TrimSpace(v.Description),
		Model:       strings.TrimSpace(v.Model),
		Status:      strings.TrimSpace(v.Status),
		IsDefault:   v.IsDefault,
		FailReason:  strings.TrimSpace(v.FailReason),
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

type qiniuAsset struct {
	ID         string `json:"qassetid"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Model      string `json:"model"`
	Status     string `json:"status"`
	GroupID    string `json:"group_id"`
	FailReason string `json:"fail_reason"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

func (v qiniuAsset) toAsset() *seedanceassets.Asset {
	return &seedanceassets.Asset{
		ID:         strings.TrimSpace(v.ID),
		GroupID:    strings.TrimSpace(v.GroupID),
		Name:       strings.TrimSpace(v.Name),
		Type:       strings.TrimSpace(v.Type),
		Model:      strings.TrimSpace(v.Model),
		Status:     strings.TrimSpace(v.Status),
		FailReason: strings.TrimSpace(v.FailReason),
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
}
