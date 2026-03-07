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

package video

import "github.com/goplus/xai/spec/kling/internal"

// ImageInput is a single image reference for multi-ref video (O1).
type ImageInput struct {
	Image string `json:"image"`
	Type  string `json:"type,omitempty"` // first_frame | end_frame
}

// VideoRef is a video reference for multi-ref video (O1).
type VideoRef struct {
	VideoURL          string `json:"video_url"`
	ReferType         string `json:"refer_type"` // feature | base
	KeepOriginalSound string `json:"keep_original_sound,omitempty"`
}

// VideoParams is the unified interface for all video model parameters.
type VideoParams interface {
	Model() string
	videoParams()
}

type videoParamsMarker struct{}

func (videoParamsMarker) videoParams() {}

// V21VideoParams for kling-v2-1 (img2video only).
type V21VideoParams struct {
	videoParamsMarker
	Prompt         string `json:"prompt"`
	InputReference string `json:"input_reference"`
	ImageTail      string `json:"image_tail,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Seconds        string `json:"seconds,omitempty"`
	Size           string `json:"size,omitempty"`
}

func (p *V21VideoParams) Model() string { return internal.ModelKlingV21Video }

// V25VideoParams for kling-v2-5-turbo (text2video + img2video).
type V25VideoParams struct {
	videoParamsMarker
	Prompt         string `json:"prompt"`
	InputReference string `json:"input_reference,omitempty"`
	ImageTail      string `json:"image_tail,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Seconds        string `json:"seconds,omitempty"`
	Size           string `json:"size,omitempty"`
}

func (p *V25VideoParams) Model() string { return internal.ModelKlingV25Turbo }

// O1VideoParams for kling-video-o1 (multi-ref).
// Uses image_list and video_list as primary; input_reference/image_tail are converted to image_list in build.
type O1VideoParams struct {
	videoParamsMarker
	Prompt         string       `json:"prompt"`
	ImageList      []ImageInput `json:"image_list,omitempty"`
	VideoList      []VideoRef   `json:"video_list,omitempty"`
	NegativePrompt string       `json:"negative_prompt,omitempty"`
	Mode           string       `json:"mode,omitempty"`
	Seconds        string       `json:"seconds,omitempty"`
	Size           string       `json:"size,omitempty"`
	VideoMode      string       `json:"video_mode,omitempty"`
}

func (p *O1VideoParams) Model() string { return internal.ModelKlingVideoO1 }

// V26VideoParams for kling-v2-6 through v2-9.
type V26VideoParams struct {
	videoParamsMarker
	ModelName             string       `json:"model"` // kling-v2-6, kling-v2-7, etc.
	Prompt                string       `json:"prompt"`
	InputReference        string       `json:"input_reference,omitempty"`
	ImageTail             string       `json:"image_tail,omitempty"`
	NegativePrompt        string       `json:"negative_prompt,omitempty"`
	Mode                  string       `json:"mode,omitempty"`
	Seconds               string       `json:"seconds,omitempty"`
	Size                  string       `json:"size,omitempty"`
	Sound                 string       `json:"sound,omitempty"`
	ImageURL              string       `json:"image_url,omitempty"`
	VideoURL              string       `json:"video_url,omitempty"`
	CharacterOrientation  string       `json:"character_orientation,omitempty"`
	KeepOriginalSound     string       `json:"keep_original_sound,omitempty"`
}

func (p *V26VideoParams) Model() string { return p.ModelName }

// V3VideoParams for kling-v3.
type V3VideoParams struct {
	videoParamsMarker
	ModelName      string `json:"model"`
	Prompt         string `json:"prompt"`
	InputReference string `json:"input_reference,omitempty"`
	Sound          string `json:"sound,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Seconds        string `json:"seconds,omitempty"`
	Size           string `json:"size,omitempty"`
}

func (p *V3VideoParams) Model() string { return p.ModelName }

// MultiPromptItem for multi_shot (V3-omni).
type MultiPromptItem struct {
	Index    int    `json:"index"`
	Prompt   string `json:"prompt"`
	Duration string `json:"duration"`
}

// V3OmniVideoParams for kling-v3-omni.
type V3OmniVideoParams struct {
	videoParamsMarker
	ModelName   string           `json:"model"`
	Prompt      string           `json:"prompt"`
	MultiShot   bool             `json:"multi_shot,omitempty"`
	ShotType    string           `json:"shot_type,omitempty"`
	MultiPrompt []MultiPromptItem `json:"multi_prompt,omitempty"`
	ImageList   []ImageInput     `json:"image_list,omitempty"`
	VideoList   []VideoRef       `json:"video_list,omitempty"`
	Sound       string           `json:"sound,omitempty"`
	Mode        string           `json:"mode,omitempty"`
	Seconds     string           `json:"seconds,omitempty"`
	Size        string           `json:"size,omitempty"`
}

func (p *V3OmniVideoParams) Model() string { return p.ModelName }
