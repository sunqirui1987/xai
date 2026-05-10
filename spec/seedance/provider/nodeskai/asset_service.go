package nodeskai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	assetPathCreateGroup = "/api/v1/digital-assets/groups/create"
	assetPathListGroups  = "/api/v1/digital-assets/groups"
	assetPathUpload      = "/api/v1/digital-assets/upload"
	assetPathGet         = "/api/v1/digital-assets/"
)

func newAssetService(client *Client) *seedanceassets.Service {
	if client == nil {
		return nil
	}
	return seedanceassets.NewWithBackend(&assetBackend{client: client})
}

type assetBackend struct {
	client *Client
}

func (b *assetBackend) CreateGroup(ctx context.Context, req *seedanceassets.CreateAssetGroupRequest) (*seedanceassets.AssetGroup, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	raw, err := b.client.PostPlatformJSON(ctx, assetPathCreateGroup, map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"description": strings.TrimSpace(req.Description),
	})
	if err != nil {
		return nil, err
	}

	var v struct {
		Success bool `json:"success"`
		Data    struct {
			ID          string `json:"Id"`
			Name        string `json:"Name"`
			Description string `json:"Description"`
			Status      string `json:"Status"`
			AssetCount  int    `json:"AssetCount"`
			CreateTime  string `json:"CreateTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("nodeskai: parse asset group create response: %w", err)
	}
	return &seedanceassets.AssetGroup{
		ID:          strings.TrimSpace(v.Data.ID),
		Name:        strings.TrimSpace(v.Data.Name),
		Description: strings.TrimSpace(v.Data.Description),
		Status:      strings.TrimSpace(v.Data.Status),
		AssetCount:  v.Data.AssetCount,
		CreateTime:  strings.TrimSpace(v.Data.CreateTime),
	}, nil
}

func (b *assetBackend) ListGroups(ctx context.Context, req *seedanceassets.ListAssetGroupsRequest) (*seedanceassets.AssetGroupList, error) {
	norm := req.Normalize()
	raw, err := b.client.PostPlatformJSON(ctx, assetPathListGroups, map[string]any{
		"name":        strings.TrimSpace(norm.Name),
		"page_number": norm.PageNumber,
		"page_size":   norm.PageSize,
	})
	if err != nil {
		return nil, err
	}

	var v struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ID          string `json:"Id"`
				Name        string `json:"Name"`
				Description string `json:"Description"`
				Status      string `json:"Status"`
				AssetCount  int    `json:"AssetCount"`
				CreateTime  string `json:"CreateTime"`
			} `json:"Items"`
			TotalCount int `json:"TotalCount"`
			PageNumber int `json:"PageNumber"`
			PageSize   int `json:"PageSize"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("nodeskai: parse asset group list response: %w", err)
	}
	items := make([]*seedanceassets.AssetGroup, 0, len(v.Data.Items))
	for _, item := range v.Data.Items {
		items = append(items, &seedanceassets.AssetGroup{
			ID:          strings.TrimSpace(item.ID),
			Name:        strings.TrimSpace(item.Name),
			Description: strings.TrimSpace(item.Description),
			Status:      strings.TrimSpace(item.Status),
			AssetCount:  item.AssetCount,
			CreateTime:  strings.TrimSpace(item.CreateTime),
		})
	}
	return &seedanceassets.AssetGroupList{
		Items:      items,
		TotalCount: v.Data.TotalCount,
		PageNumber: v.Data.PageNumber,
		PageSize:   v.Data.PageSize,
	}, nil
}

func (b *assetBackend) UploadAsset(ctx context.Context, req *seedanceassets.UploadAssetRequest) (*seedanceassets.AssetUploadResult, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(req.File)
	if err != nil {
		return nil, fmt.Errorf("nodeskai: read asset file: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(req.FileName))
	if err != nil {
		return nil, fmt.Errorf("nodeskai: create asset multipart file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("nodeskai: write asset multipart file: %w", err)
	}
	if err := writer.WriteField("group_id", req.GroupID); err != nil {
		return nil, fmt.Errorf("nodeskai: write asset group_id: %w", err)
	}
	if assetType := strings.TrimSpace(req.AssetType); assetType != "" {
		if err := writer.WriteField("asset_type", assetType); err != nil {
			return nil, fmt.Errorf("nodeskai: write asset_type: %w", err)
		}
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		if err := writer.WriteField("name", name); err != nil {
			return nil, fmt.Errorf("nodeskai: write asset name: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("nodeskai: finalize asset multipart body: %w", err)
	}

	raw, err := b.client.PostMultipart(ctx, assetPathUpload, writer.FormDataContentType(), body.Bytes())
	if err != nil {
		return nil, err
	}

	var v struct {
		Success   bool   `json:"success"`
		AssetID   string `json:"asset_id"`
		AssetType string `json:"asset_type"`
		Status    string `json:"status"`
		TOSURL    string `json:"tos_url"`
		FileName  string `json:"file_name"`
		FileSize  int64  `json:"file_size"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("nodeskai: parse asset upload response: %w", err)
	}
	return &seedanceassets.AssetUploadResult{
		Success:   v.Success,
		AssetID:   strings.TrimSpace(v.AssetID),
		AssetType: strings.TrimSpace(v.AssetType),
		Status:    strings.TrimSpace(v.Status),
		TOSURL:    strings.TrimSpace(v.TOSURL),
		FileName:  strings.TrimSpace(v.FileName),
		FileSize:  v.FileSize,
	}, nil
}

func (b *assetBackend) GetAsset(ctx context.Context, assetID string) (*seedanceassets.Asset, error) {
	raw, err := b.client.PostPlatformJSON(ctx, assetPathGet+strings.TrimSpace(assetID), nil)
	if err != nil {
		return nil, err
	}

	var v struct {
		Success bool `json:"success"`
		Data    struct {
			ID         string `json:"Id"`
			GroupID    string `json:"GroupId"`
			Name       string `json:"Name"`
			Type       string `json:"Type"`
			AssetType  string `json:"AssetType"`
			Status     string `json:"Status"`
			URL        string `json:"Url"`
			URLUpper   string `json:"URL"`
			CreateTime string `json:"CreateTime"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("nodeskai: parse asset get response: %w", err)
	}

	assetType := strings.TrimSpace(v.Data.Type)
	if assetType == "" {
		assetType = strings.TrimSpace(v.Data.AssetType)
	}
	assetURL := strings.TrimSpace(v.Data.URL)
	if assetURL == "" {
		assetURL = strings.TrimSpace(v.Data.URLUpper)
	}
	return &seedanceassets.Asset{
		ID:         strings.TrimSpace(v.Data.ID),
		GroupID:    strings.TrimSpace(v.Data.GroupID),
		Name:       strings.TrimSpace(v.Data.Name),
		Type:       assetType,
		Status:     strings.TrimSpace(v.Data.Status),
		URL:        assetURL,
		CreateTime: strings.TrimSpace(v.Data.CreateTime),
	}, nil
}

func newAssetPlatformClient(videoClient *Client) *Client {
	if videoClient == nil {
		return nil
	}
	videoClientID := strings.TrimSpace(videoClient.clientID)
	videoClientSecret := strings.TrimSpace(videoClient.clientSecret)
	envClientID := strings.TrimSpace(firstNonEmptyEnv("NODESKAI_CLIENT_ID", "NODESK_CLIENT_ID"))
	envClientSecret := strings.TrimSpace(firstNonEmptyEnv("NODESKAI_CLIENT_SECRET", "NODESK_CLIENT_SECRET"))

	videoClient.LogDebug("newAssetPlatformClient credential sources video_client_id=%v video_client_secret=%v env_client_id=%v env_client_secret=%v",
		videoClientID != "",
		videoClientSecret != "",
		envClientID != "",
		envClientSecret != "",
	)

	clientID := videoClientID
	if clientID == "" {
		clientID = envClientID
	}
	clientSecret := videoClientSecret
	if clientSecret == "" {
		clientSecret = envClientSecret
	}
	if clientID == "" || clientSecret == "" {
		videoClient.LogDebug("newAssetPlatformClient missing credentials after merge client_id=%v client_secret=%v",
			clientID != "",
			clientSecret != "",
		)
		return nil
	}
	videoClient.LogDebug("newAssetPlatformClient ready client_id=%v client_secret=%v",
		clientID != "",
		clientSecret != "",
	)
	return &Client{
		httpClient:      videoClient.httpClient,
		baseURL:         videoClient.baseURL,
		platformBaseURL: videoClient.platformBaseURL,
		externalUserID:  videoClient.externalUserID,
		clientID:        clientID,
		clientSecret:    clientSecret,
		maxRetries:      videoClient.maxRetries,
		baseRetryDelay:  videoClient.baseRetryDelay,
		debugLog:        videoClient.debugLog,
		logger:          videoClient.logger,
	}
}
