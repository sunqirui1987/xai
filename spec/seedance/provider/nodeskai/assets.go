package nodeskai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/goplus/xai/spec/seedance"
	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
)

const (
	defaultAssetGroupName = "默认素材组"
	defaultAssetGroupDesc = "Seedance 2.0 默认素材组"
)

func (b *backend) prepareParams(ctx context.Context, p *seedance.Params) (*seedance.Params, error) {
	groupID, err := b.resolveAssetGroupID(ctx, p)
	if err != nil {
		return nil, err
	}
	if groupID == "" {
		return p, nil
	}

	out := seedance.NewParams()
	for k, v := range p.Export() {
		out.Set(k, v)
	}

	if raw, ok := p.Get(ParamContent); ok {
		content := normalizeContent(raw)
		if len(content) == 0 {
			return nil, fmt.Errorf("nodeskai: content is empty")
		}
		resolved, err := b.resolveContentImages(ctx, content, groupID)
		if err != nil {
			return nil, err
		}
		out.Set(ParamContent, resolved)
		return out, nil
	}

	refs := p.GetReferenceImages(seedance.ParamReferenceImages)
	if len(refs) > 0 {
		items := make([]map[string]any, 0, len(refs))
		for _, ref := range refs {
			resolvedURL, err := b.ensureAssetURL(ctx, ref.URL, groupID)
			if err != nil {
				return nil, err
			}
			items = append(items, map[string]any{
				"url":  resolvedURL,
				"role": ref.Role,
			})
		}
		out.Set(seedance.ParamReferenceImages, items)
	}

	refURLs := p.GetStringSlice(seedance.ParamReferenceImageURLs)
	if len(refURLs) > 0 {
		resolved := make([]string, 0, len(refURLs))
		for _, refURL := range refURLs {
			u, err := b.ensureAssetURL(ctx, refURL, groupID)
			if err != nil {
				return nil, err
			}
			resolved = append(resolved, u)
		}
		out.Set(seedance.ParamReferenceImageURLs, resolved)
	}

	return out, nil
}

func (b *backend) resolveAssetGroupID(ctx context.Context, p *seedance.Params) (string, error) {
	groupID := strings.TrimSpace(firstNonEmptyParamOrEnv(p, ParamAssetGroupID, "NODESKAI_ASSET_GROUP_ID"))
	if groupID != "" {
		return groupID, nil
	}
	if !paramsContainReferenceImages(p) {
		return "", nil
	}
	if b.assetService == nil {
		return "", fmt.Errorf("nodeskai: image asset flow requires NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET")
	}

	groupName := strings.TrimSpace(os.Getenv("NODESKAI_DEFAULT_ASSET_GROUP_NAME"))
	if groupName == "" {
		groupName = defaultAssetGroupName
	}
	group, err := b.findAssetGroupByName(ctx, groupName)
	if err != nil {
		return "", err
	}
	if group != nil && strings.TrimSpace(group.ID) != "" {
		return group.ID, nil
	}

	created, err := b.assetService.CreateGroup(ctx, &seedanceassets.CreateAssetGroupRequest{
		Name:        groupName,
		Description: defaultAssetGroupDesc,
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func paramsContainReferenceImages(p *seedance.Params) bool {
	if p == nil {
		return false
	}
	if len(p.GetReferenceImages(seedance.ParamReferenceImages)) > 0 {
		return true
	}
	if len(p.GetStringSlice(seedance.ParamReferenceImageURLs)) > 0 {
		return true
	}
	if raw, ok := p.Get(ParamContent); ok {
		for _, item := range normalizeContent(raw) {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType, _ := m["type"].(string)
			if strings.TrimSpace(itemType) == "image_url" {
				return true
			}
		}
	}
	return false
}

func (b *backend) resolveContentImages(ctx context.Context, content []any, groupID string) ([]any, error) {
	out := make([]any, 0, len(content))
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		itemType, _ := m["type"].(string)
		if strings.TrimSpace(itemType) != "image_url" {
			out = append(out, item)
			continue
		}
		payload, _ := m["image_url"].(map[string]any)
		if payload == nil {
			out = append(out, item)
			continue
		}
		rawURL, _ := payload["url"].(string)
		if strings.TrimSpace(rawURL) == "" {
			out = append(out, item)
			continue
		}

		resolvedURL, err := b.ensureAssetURL(ctx, rawURL, groupID)
		if err != nil {
			return nil, err
		}

		cloned := make(map[string]any, len(m))
		for k, v := range m {
			cloned[k] = v
		}
		imageURL := make(map[string]any, len(payload))
		for k, v := range payload {
			imageURL[k] = v
		}
		imageURL["url"] = resolvedURL
		cloned["image_url"] = imageURL
		out = append(out, cloned)
	}
	return out, nil
}

func (b *backend) ensureAssetURL(ctx context.Context, rawURL, groupID string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	groupID = strings.TrimSpace(groupID)
	if rawURL == "" || groupID == "" {
		return rawURL, nil
	}
	if b.assetService == nil {
		return "", fmt.Errorf("nodeskai: image asset flow requires NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET")
	}

	fileName, body, err := b.downloadImage(ctx, rawURL)
	if err != nil {
		return "", err
	}
	ref, err := b.assetService.UploadAndAwaitAsset(ctx, &seedanceassets.UploadAssetRequest{
		GroupID:  groupID,
		Name:     safeAssetName(fileName),
		FileName: fileName,
		File:     bytes.NewReader(body),
	}, 2*time.Second)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(ref.Asset) == "" {
		return "", fmt.Errorf("nodeskai: asset ref is empty after activation")
	}
	return ref.Asset, nil
}

func (b *backend) downloadImage(ctx context.Context, rawURL string) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("nodeskai: create image download request: %w", err)
	}
	resp, err := b.client.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("nodeskai: download image %q: %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("nodeskai: download image %q HTTP %d", rawURL, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("nodeskai: read image %q: %w", rawURL, err)
	}
	return inferAssetFileName(rawURL, resp.Header.Get("Content-Type")), data, nil
}

func inferAssetFileName(rawURL, contentType string) string {
	if u, err := url.Parse(rawURL); err == nil {
		if base := path.Base(strings.TrimSpace(u.Path)); base != "" && base != "." && base != "/" {
			return base
		}
	}
	exts, _ := mime.ExtensionsByType(strings.TrimSpace(contentType))
	if len(exts) > 0 {
		return "reference" + exts[0]
	}
	return "reference_image"
}

func safeAssetName(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return "参考图"
	}
	ext := filepath.Ext(name)
	base := strings.TrimSpace(strings.TrimSuffix(name, ext))
	if base == "" {
		base = "参考图"
	}
	runes := []rune(base)
	if len(runes) > 12 {
		base = string(runes[:12])
	}
	if strings.TrimSpace(base) == "" {
		return "参考图"
	}
	return base
}

func firstNonEmptyParamOrEnv(p *seedance.Params, paramName string, envName string) string {
	if p != nil {
		if v := strings.TrimSpace(p.GetString(paramName)); v != "" {
			return v
		}
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (b *backend) findAssetGroupByName(ctx context.Context, name string) (*seedanceassets.AssetGroup, error) {
	if b.assetService == nil {
		return nil, fmt.Errorf("nodeskai: image asset flow requires NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET")
	}
	ret, err := b.assetService.ListGroups(ctx, &seedanceassets.ListAssetGroupsRequest{
		Name:       name,
		PageNumber: 1,
		PageSize:   20,
	})
	if err != nil {
		return nil, err
	}
	target := strings.TrimSpace(name)
	for _, item := range ret.Items {
		if item == nil || strings.TrimSpace(item.Name) != target {
			continue
		}
		return item, nil
	}
	return nil, nil
}
