/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package seedance

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	xai "github.com/goplus/xai/spec"
)

// Param name constants (Ark contents/generations/tasks body).
const (
	ParamText               = "text"
	ParamPrompt             = "prompt" // alias → first text block
	ParamReferenceImageURLs = "reference_image_urls"
	ParamReferenceVideoURLs = "reference_video_urls"
	ParamReferenceAudioURLs = "reference_audio_urls"
	ParamDuration           = "duration"
	ParamRatio              = "ratio"
	ParamGenerateAudio      = "generate_audio"
	ParamWatermark          = "watermark"

	// ParamArkContentJSON is optional raw JSON array for the Ark request field "content".
	// When non-empty, it replaces the synthesized content from text / reference_*_urls
	// (see [Volc Ark 创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh)).
	ParamArkContentJSON = "ark_content_json"
)

var (
	ErrTextRequired = errors.New("seedance: text or prompt is required")
)

// Params stores GenVideo inputs. Field names match xai.Operation InputSchema and map to Ark JSON
// (see provider/volc buildTaskBody). Use ParamArkContentJSON for a raw content[] array.
type Params struct {
	m map[string]any
}

// NewParams creates an empty Params.
func NewParams() *Params {
	return &Params{m: make(map[string]any)}
}

// Set implements xai.Params.
func (p *Params) Set(name string, val any) xai.Params {
	p.m[name] = val
	return p
}

// Export returns a shallow copy for backends.
func (p *Params) Export() map[string]any {
	out := make(map[string]any, len(p.m))
	for k, v := range p.m {
		out[k] = v
	}
	return out
}

// Get returns a raw value.
func (p *Params) Get(name string) (any, bool) {
	v, ok := p.m[name]
	return v, ok
}

// PrimaryText returns the main script: ParamText, or ParamPrompt, or empty.
func (p *Params) PrimaryText() string {
	if s := strings.TrimSpace(p.GetString(ParamText)); s != "" {
		return s
	}
	return strings.TrimSpace(p.GetString(ParamPrompt))
}

// ArkContentFromJSON returns the full Ark "content" array when ParamArkContentJSON is set and valid.
func (p *Params) ArkContentFromJSON() ([]any, error) {
	raw := strings.TrimSpace(p.GetString(ParamArkContentJSON))
	if raw == "" {
		return nil, nil
	}
	var items []any
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items, nil
}

// GetString returns a trimmed string or "".
func (p *Params) GetString(name string) string {
	v, ok := p.m[name]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// GetInt returns *int or nil.
func (p *Params) GetInt(name string) *int {
	v, ok := p.m[name]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case int:
		return &x
	case int64:
		i := int(x)
		return &i
	case float64:
		i := int(x)
		return &i
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		return &n
	default:
		return nil
	}
}

// GetBool returns *bool or nil.
func (p *Params) GetBool(name string) *bool {
	v, ok := p.m[name]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case bool:
		return &x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		switch s {
		case "true", "1", "yes", "on":
			b := true
			return &b
		case "false", "0", "no", "off":
			b := false
			return &b
		default:
			return nil
		}
	default:
		return nil
	}
}

// GetStringSlice parses []string, single string, or []any of strings.
func (p *Params) GetStringSlice(name string) []string {
	v, ok := p.m[name]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []string{s}
	case []string:
		return sanitizeStrings(x)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := item.(string)
			if ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

func sanitizeStrings(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
