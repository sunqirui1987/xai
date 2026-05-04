/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package qiniu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
)

const pathCreateTask = "/v3/contents/generations/tasks"

// ErrTaskFailed is returned when Qiniu reports a terminal failure.
var ErrTaskFailed = errors.New("qiniu-seedance: task failed")

type backend struct {
	client *Client
}

func newBackend(client *Client) *backend {
	return &backend{client: client}
}

// NewBackend returns a seedance.Backend backed by the Qiniu/Qnagic HTTP API.
func NewBackend(client *Client) seedance.Backend {
	return newBackend(client)
}

// Submit creates an async generation task and returns a polling OperationResponse.
func (b *backend) Submit(ctx context.Context, model xai.Model, params xai.Params) (xai.OperationResponse, error) {
	p, ok := params.(*seedance.Params)
	if !ok {
		return nil, fmt.Errorf("qiniu-seedance: expected *seedance.Params, got %T", params)
	}
	m := strings.TrimSpace(string(model))
	b.client.LogDebug("Submit GenVideo model=%q", m)

	body, err := buildTaskBody(m, p)
	if err != nil {
		b.client.LogDebug("Submit buildTaskBody error: %v", err)
		return nil, err
	}
	raw, err := b.client.PostJSON(ctx, pathCreateTask, body)
	if err != nil {
		b.client.LogDebug("Submit POST error: %v", err)
		return nil, err
	}
	taskID, err := parseCreateTaskID(raw)
	if err != nil {
		b.client.LogDebug("Submit parse task id error: %v", err)
		return nil, err
	}
	b.client.LogDebug("Submit created task_id=%q (async polling)", taskID)
	return b.newPollingResponse(taskID), nil
}

// GetTaskStatus loads task state and converts terminal success/failure into xai.OperationResponse.
func (b *backend) GetTaskStatus(ctx context.Context, taskID string) (xai.OperationResponse, error) {
	b.client.LogDebug("GetTaskStatus task_id=%q", taskID)
	raw, err := b.getTaskJSON(ctx, taskID)
	if err != nil {
		b.client.LogDebug("GetTaskStatus request error: %v", err)
		return nil, err
	}
	status, videoURL, failMsg, err := parseTaskGetResponse(raw)
	if err != nil {
		b.client.LogDebug("GetTaskStatus parse error: %v", err)
		return nil, err
	}
	st := normalizeStatus(status)
	b.client.LogDebug("GetTaskStatus task_id=%q raw_status=%q normalized=%q video_url_present=%v", taskID, status, st, videoURL != "")

	switch {
	case isSuccessStatus(st):
		if videoURL == "" {
			return nil, fmt.Errorf("%w: succeeded but no video_url in response", ErrTaskFailed)
		}
		return &seedance.SyncOperationResponse{R: seedance.NewOutputVideos([]string{videoURL})}, nil
	case isFailedStatus(st):
		if failMsg == "" {
			failMsg = "unknown error"
		}
		return nil, fmt.Errorf("%w: %s", ErrTaskFailed, failMsg)
	default:
		return b.newPollingResponse(taskID), nil
	}
}

func (b *backend) newPollingResponse(taskID string) xai.OperationResponse {
	return seedance.NewAsyncOperationResponse(func(ctx context.Context) (xai.OperationResponse, error) {
		return b.GetTaskStatus(ctx, taskID)
	}, taskID)
}

func (b *backend) getTaskJSON(ctx context.Context, taskID string) ([]byte, error) {
	path := pathCreateTask + "/" + url.PathEscape(taskID)
	return b.client.GetJSON(ctx, path)
}

func buildTaskBody(model string, p *seedance.Params) (map[string]any, error) {
	m := normalizeQiniuModel(model)
	if m == "" {
		return nil, fmt.Errorf("qiniu-seedance: empty model")
	}

	text := p.PrimaryText()
	if text == "" {
		return nil, seedance.ErrTextRequired
	}

	content := []any{
		map[string]any{
			"type": "text",
			"text": text,
		},
	}

	explicitRefs := p.GetReferenceImages(seedance.ParamReferenceImages)
	if len(explicitRefs) > 0 {
		for _, ref := range explicitRefs {
			role := strings.TrimSpace(ref.Role)
			if role == "" {
				role = "reference_image"
			}
			content = append(content, imageURLContent(ref.URL, role))
		}
	} else {
		for _, refURL := range p.GetStringSlice(seedance.ParamReferenceImageURLs) {
			content = append(content, imageURLContent(refURL, "reference_image"))
		}
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceVideoURLs) {
		content = append(content, mediaURLContent("video_url", "video_url", refURL, "reference_video"))
	}
	for _, refURL := range p.GetStringSlice(seedance.ParamReferenceAudioURLs) {
		content = append(content, mediaURLContent("audio_url", "audio_url", refURL, "reference_audio"))
	}

	body := map[string]any{
		"model":   m,
		"content": content,
	}

	if d := p.GetInt(seedance.ParamDuration); d != nil {
		if err := seedance.ValidateVideoDuration(m, *d); err != nil {
			return nil, err
		}
		body["duration"] = *d
	}
	if r := p.GetString(seedance.ParamRatio); r != "" {
		body["ratio"] = r
	}
	if ga := p.GetBool(seedance.ParamGenerateAudio); ga != nil {
		body["generate_audio"] = *ga
	}
	return body, nil
}

func normalizeQiniuModel(model string) string {
	m := strings.TrimSpace(model)
	if m == "" {
		return ""
	}
	if strings.Contains(m, "/") {
		return m
	}
	if strings.HasPrefix(strings.ToLower(m), "doubao-seedance-") {
		return "bytedance/" + m
	}
	return m
}

func imageURLContent(u string, role string) map[string]any {
	return mediaURLContent("image_url", "image_url", u, role)
}

func mediaURLContent(contentType string, field string, u string, role string) map[string]any {
	return map[string]any{
		"type": contentType,
		field: map[string]any{
			"url": u,
		},
		"role": role,
	}
}

func parseCreateTaskID(raw []byte) (string, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("qiniu-seedance: parse create response: %w", err)
	}
	if id := stringVal(v, "id"); id != "" {
		return id, nil
	}
	if d, ok := v["data"].(map[string]any); ok {
		if id := stringVal(d, "id"); id != "" {
			return id, nil
		}
		if id := stringVal(d, "task_id"); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("qiniu-seedance: no task id in create response")
}

func parseTaskGetResponse(raw []byte) (status, videoURL, failMsg string, err error) {
	var v map[string]any
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", "", "", fmt.Errorf("qiniu-seedance: parse get task response: %w", err)
	}
	status = firstStatus(v)
	videoURL = findVideoURL(v)
	failMsg = firstErrorMessage(v)
	if d, ok := v["data"].(map[string]any); ok {
		if status == "" {
			status = firstStatus(d)
		}
		if videoURL == "" {
			videoURL = findVideoURL(d)
		}
		if failMsg == "" {
			failMsg = firstErrorMessage(d)
		}
	}
	return status, videoURL, failMsg, nil
}

func firstStatus(m map[string]any) string {
	return stringVal(m, "status")
}

func firstErrorMessage(m map[string]any) string {
	if s := stringVal(m, "message"); s != "" {
		return s
	}
	if s := stringVal(m, "error"); s != "" {
		return s
	}
	if em, ok := m["error"].(map[string]any); ok {
		return stringVal(em, "message")
	}
	return ""
}

func findVideoURL(m map[string]any) string {
	if s := stringVal(m, "video_url"); s != "" {
		return s
	}
	if c, ok := m["content"].(map[string]any); ok {
		if s := stringVal(c, "video_url"); s != "" {
			return s
		}
		if nested, ok := c["video_url"].(map[string]any); ok {
			if s := stringVal(nested, "url"); s != "" {
				return s
			}
		}
	}
	if arr, ok := m["content"].([]any); ok {
		for _, item := range arr {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if s := stringVal(im, "video_url"); s != "" {
				return s
			}
			if nested, ok := im["video_url"].(map[string]any); ok {
				if s := stringVal(nested, "url"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func stringVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func normalizeStatus(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isSuccessStatus(s string) bool {
	switch s {
	case "succeeded", "success", "completed", "complete", "done":
		return true
	default:
		return false
	}
}

func isFailedStatus(s string) bool {
	switch s {
	case "failed", "error", "cancelled", "canceled", "expired":
		return true
	default:
		return false
	}
}
