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

package kling

import (
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling/internal"
	"github.com/goplus/xai/spec/kling/video"
)

// Re-export video validation errors for backward compatibility.
var (
	ErrInputReferenceRequired  = video.ErrInputReferenceRequired
	ErrKeyframeModeRequired    = video.ErrKeyframeModeRequired
	ErrKeyframeSecondsRequired = video.ErrKeyframeSecondsRequired
)

// -----------------------------------------------------------------------------
// Param name constants (re-exported from internal)
// -----------------------------------------------------------------------------

const (
	ParamPrompt               = internal.ParamPrompt
	ParamAspectRatio          = internal.ParamAspectRatio
	ParamReferenceImages      = internal.ParamReferenceImages
	ParamImage                = internal.ParamImage
	ParamImageReference       = internal.ParamImageReference
	ParamNegativePrompt       = internal.ParamNegativePrompt
	ParamImageFidelity        = internal.ParamImageFidelity
	ParamHumanFidelity        = internal.ParamHumanFidelity
	ParamN                    = internal.ParamN
	ParamResolution           = internal.ParamResolution
	ParamInputReference       = internal.ParamInputReference
	ParamImageTail            = internal.ParamImageTail
	ParamMode                 = internal.ParamMode
	ParamSeconds              = internal.ParamSeconds
	ParamSize                 = internal.ParamSize
	ParamImageList            = internal.ParamImageList
	ParamVideoList            = internal.ParamVideoList
	ParamVideoMode            = internal.ParamVideoMode
	ParamSound                = internal.ParamSound
	ParamSubjectImageList     = internal.ParamSubjectImageList
	ParamSubjectImage         = internal.ParamSubjectImage
	ParamSceneImage           = internal.ParamSceneImage
	ParamStyleImage           = internal.ParamStyleImage
	ParamImageUrl             = internal.ParamImageUrl
	ParamVideoUrl             = internal.ParamVideoUrl
	ParamCharacterOrientation = internal.ParamCharacterOrientation
	ParamKeepOriginalSound    = internal.ParamKeepOriginalSound
	ParamMultiShot            = internal.ParamMultiShot
	ParamShotType             = internal.ParamShotType
	ParamMultiPrompt          = internal.ParamMultiPrompt
)

// -----------------------------------------------------------------------------
// Optional video param values (use with ParamSize, ParamMode, ParamSeconds)
// -----------------------------------------------------------------------------

// Video size options (re-exported from internal)
const (
	Size1920x1080 = internal.Size1920x1080 // 横屏 16:9
	Size1080x1920 = internal.Size1080x1920 // 竖屏 9:16
	Size1280x720  = internal.Size1280x720  // 横屏 16:9
	Size720x1280  = internal.Size720x1280  // 竖屏 9:16
	Size1080x1080 = internal.Size1080x1080 // 方形 1:1
	Size720x720   = internal.Size720x720   // 方形 1:1
)

// Video mode options
const (
	ModeStd = internal.ModeStd // 标准模式 720P
	ModePro = internal.ModePro // 专家模式 1080P
)

// Video seconds options
const (
	Seconds5  = internal.Seconds5
	Seconds10 = internal.Seconds10
)

// Video sound options (V2.6+, V3, V3-omni)
const (
	SoundOn  = internal.SoundOn
	SoundOff = internal.SoundOff
)

// -----------------------------------------------------------------------------
// Image O1 options (kling-image-o1: resolution, aspect_ratio, num_images 1-9)
// -----------------------------------------------------------------------------

// Resolution options (kling-image-o1 only, default 1K)
const (
	Resolution1K = internal.Resolution1K
	Resolution2K = internal.Resolution2K
)

// AspectRatio options (kling-image-o1 default auto; other models vary)
const (
	AspectAuto  = internal.AspectAuto
	Aspect16x9  = internal.Aspect16x9
	Aspect9x16  = internal.Aspect9x16
	Aspect1x1   = internal.Aspect1x1
	Aspect4x3   = internal.Aspect4x3
	Aspect3x4   = internal.Aspect3x4
	Aspect3x2   = internal.Aspect3x2
	Aspect2x3   = internal.Aspect2x3
	Aspect21x9  = internal.Aspect21x9
	// Image reference (kling-v1-5)
	ImageRefSubject = internal.ImageRefSubject
	ImageRefFace    = internal.ImageRefFace
)

// -----------------------------------------------------------------------------

// Params implements xai.Params with map-based storage. Used by genImage and genVideo
// operations. Export() is for internal use by Executor implementations.
// Params implements video.ParamsChecker for validation and internal.ParamsReader for build.
type Params struct {
	m map[string]any
}

// NewParams creates a new Params instance.
func NewParams() *Params {
	return &Params{m: make(map[string]any)}
}

// Set sets a parameter. Implements xai.Params.
func (p *Params) Set(name string, val any) xai.Params {
	p.m[name] = val
	return p
}

// Export returns the params as a map for Executor implementations. Internal use only.
func (p *Params) Export() map[string]any {
	out := make(map[string]any, len(p.m))
	for k, v := range p.m {
		out[k] = v
	}
	return out
}

// Get returns the raw value for the given param name and whether it exists.
func (p *Params) Get(name string) (any, bool) {
	v, ok := p.m[name]
	return v, ok
}

// HasNonEmptyString returns true if the param exists and is a non-empty string.
// Implements video.ParamsChecker.
func (p *Params) HasNonEmptyString(name string) bool {
	v, ok := p.m[name]
	if !ok {
		return false
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) != ""
}

// GetString returns the trimmed string value of the param, or "" if missing or not a string.
// Implements video.ParamsChecker.
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
