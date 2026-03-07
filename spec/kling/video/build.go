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
 * WITHOUT WARRANTIES OR CONDITIONS OF KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package video

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goplus/xai/spec/kling/internal"
)

// BuildVideoParams builds typed VideoParams from ParamsReader for the given model.
func BuildVideoParams(model string, p internal.ParamsReader) (VideoParams, error) {
	if p == nil {
		return nil, errors.New("kling: params is nil")
	}
	m := strings.ToLower(model)
	prompt := p.GetString(internal.ParamPrompt)
	if strings.TrimSpace(prompt) == "" {
		return nil, internal.ErrPromptRequired
	}

	switch {
	case m == internal.ModelKlingV21Video:
		return buildV21VideoParams(model, prompt, p), nil
	case m == internal.ModelKlingV25Turbo:
		return buildV25VideoParams(model, prompt, p), nil
	case m == internal.ModelKlingVideoO1:
		return buildO1VideoParams(model, prompt, p), nil
	case m == internal.ModelKlingV26 || m == internal.ModelKlingV27 ||
		m == internal.ModelKlingV28 || m == internal.ModelKlingV29:
		return buildV26VideoParams(model, prompt, p), nil
	case m == internal.ModelKlingV3:
		return buildV3VideoParams(model, prompt, p), nil
	case m == internal.ModelKlingV3Omni:
		return buildV3OmniVideoParams(model, prompt, p), nil
	default:
		return nil, fmt.Errorf("kling: unsupported video model %q", model)
	}
}

func buildV21VideoParams(model, prompt string, p internal.ParamsReader) *V21VideoParams {
	return &V21VideoParams{
		Prompt:         prompt,
		InputReference: p.GetString(internal.ParamInputReference),
		ImageTail:      p.GetString(internal.ParamImageTail),
		NegativePrompt: p.GetString(internal.ParamNegativePrompt),
		Mode:           p.GetString(internal.ParamMode),
		Seconds:        p.GetString(internal.ParamSeconds),
		Size:           p.GetString(internal.ParamSize),
	}
}

func buildV25VideoParams(model, prompt string, p internal.ParamsReader) *V25VideoParams {
	return &V25VideoParams{
		Prompt:         prompt,
		InputReference: p.GetString(internal.ParamInputReference),
		ImageTail:      p.GetString(internal.ParamImageTail),
		NegativePrompt: p.GetString(internal.ParamNegativePrompt),
		Mode:           p.GetString(internal.ParamMode),
		Seconds:        p.GetString(internal.ParamSeconds),
		Size:           p.GetString(internal.ParamSize),
	}
}

func buildO1VideoParams(model, prompt string, p internal.ParamsReader) *O1VideoParams {
	imgList := getImageList(p)
	if len(imgList) == 0 {
		inputRef := p.GetString(internal.ParamInputReference)
		if inputRef != "" {
			imgList = append(imgList, ImageInput{Image: inputRef, Type: "first_frame"})
			imgTail := p.GetString(internal.ParamImageTail)
			if imgTail != "" {
				imgList = append(imgList, ImageInput{Image: imgTail, Type: "end_frame"})
			}
		}
	}
	return &O1VideoParams{
		Prompt:         prompt,
		ImageList:      imgList,
		VideoList:      getVideoList(p),
		NegativePrompt: p.GetString(internal.ParamNegativePrompt),
		Mode:           p.GetString(internal.ParamMode),
		Seconds:        p.GetString(internal.ParamSeconds),
		Size:           p.GetString(internal.ParamSize),
		VideoMode:      p.GetString(internal.ParamVideoMode),
	}
}

func buildV3VideoParams(model, prompt string, p internal.ParamsReader) *V3VideoParams {
	return &V3VideoParams{
		ModelName:      model,
		Prompt:         prompt,
		InputReference: p.GetString(internal.ParamInputReference),
		Sound:          p.GetString(internal.ParamSound),
		Mode:           p.GetString(internal.ParamMode),
		Seconds:        p.GetString(internal.ParamSeconds),
		Size:           p.GetString(internal.ParamSize),
	}
}

func buildV3OmniVideoParams(model, prompt string, p internal.ParamsReader) *V3OmniVideoParams {
	return &V3OmniVideoParams{
		ModelName:   model,
		Prompt:      prompt,
		MultiShot:   internal.GetBool(p, internal.ParamMultiShot),
		ShotType:    p.GetString(internal.ParamShotType),
		MultiPrompt: getMultiPrompt(p),
		ImageList:   getImageList(p),
		VideoList:   getVideoList(p),
		Sound:       p.GetString(internal.ParamSound),
		Mode:        p.GetString(internal.ParamMode),
		Seconds:     p.GetString(internal.ParamSeconds),
		Size:        p.GetString(internal.ParamSize),
	}
}

func getMultiPrompt(p internal.ParamsReader) []MultiPromptItem {
	v, ok := p.Get(internal.ParamMultiPrompt)
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []MultiPromptItem
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		mp := MultiPromptItem{}
		if i, ok := m["index"].(int); ok {
			mp.Index = i
		} else if f, ok := m["index"].(float64); ok {
			mp.Index = int(f)
		}
		if s, ok := m["prompt"].(string); ok {
			mp.Prompt = s
		}
		if s, ok := m["duration"].(string); ok {
			mp.Duration = s
		}
		out = append(out, mp)
	}
	return out
}

func buildV26VideoParams(model, prompt string, p internal.ParamsReader) *V26VideoParams {
	return &V26VideoParams{
		ModelName:            model,
		Prompt:               prompt,
		InputReference:       p.GetString(internal.ParamInputReference),
		ImageTail:            p.GetString(internal.ParamImageTail),
		NegativePrompt:       p.GetString(internal.ParamNegativePrompt),
		Mode:                 p.GetString(internal.ParamMode),
		Seconds:              p.GetString(internal.ParamSeconds),
		Size:                 p.GetString(internal.ParamSize),
		Sound:                p.GetString(internal.ParamSound),
		ImageURL:             p.GetString(internal.ParamImageUrl),
		VideoURL:             p.GetString(internal.ParamVideoUrl),
		CharacterOrientation: p.GetString(internal.ParamCharacterOrientation),
		KeepOriginalSound:    p.GetString(internal.ParamKeepOriginalSound),
	}
}

func getImageList(p internal.ParamsReader) []ImageInput {
	v, ok := p.Get(internal.ParamImageList)
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []ImageInput
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		img := ImageInput{}
		if s, ok := m["image"].(string); ok {
			img.Image = s
		}
		if s, ok := m["type"].(string); ok {
			img.Type = s
		}
		if img.Image != "" {
			out = append(out, img)
		}
	}
	return out
}

func getVideoList(p internal.ParamsReader) []VideoRef {
	v, ok := p.Get(internal.ParamVideoList)
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []VideoRef
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		ref := VideoRef{}
		if s, ok := m["video_url"].(string); ok {
			ref.VideoURL = s
		}
		if s, ok := m["refer_type"].(string); ok {
			ref.ReferType = s
		}
		if s, ok := m["keep_original_sound"].(string); ok {
			ref.KeepOriginalSound = s
		}
		if ref.VideoURL != "" {
			out = append(out, ref)
		}
	}
	return out
}
