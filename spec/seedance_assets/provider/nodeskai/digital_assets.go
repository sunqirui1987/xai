/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 */

package nodeskai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	seedancenodeskai "github.com/goplus/xai/spec/seedance/provider/nodeskai"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

func createGroup(ctx context.Context, client *seedancenodeskai.Client, req *seedanceassets.CreateAssetGroupRequest) (*seedanceassets.AssetGroup, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	raw, err := client.PostPlatformJSON(ctx, pathCreateGroup, map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"description": strings.TrimSpace(req.Description),
	})
	if err != nil {
		return nil, err
	}
	return parseCreateGroupResponse(raw)
}

func uploadAsset(ctx context.Context, client *seedancenodeskai.Client, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(req.File)
	if err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: read asset file: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(req.FileName))
	if err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: create multipart file field: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: write multipart file field: %w", err)
	}
	if err := writer.WriteField("group_id", req.GroupID); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: write group_id: %w", err)
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		if err := writer.WriteField("name", name); err != nil {
			return nil, fmt.Errorf("seedance_assets/nodeskai: write name: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("seedance_assets/nodeskai: finalize multipart body: %w", err)
	}

	raw, err := client.PostMultipart(ctx, pathUploadAsset, writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return nil, err
	}
	return parseUploadAssetResponse(raw)
}

func getAsset(ctx context.Context, client *seedancenodeskai.Client, assetID string) (*seedanceassets.Asset, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil, fmt.Errorf("seedance_assets/nodeskai: asset_id is required")
	}
	raw, err := client.PostPlatformJSON(ctx, pathGetAsset+assetID, nil)
	if err != nil {
		return nil, err
	}
	return parseGetAssetResponse(raw)
}
