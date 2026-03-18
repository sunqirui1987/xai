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

package gemini

import (
	"fmt"
	"strings"

	xai "github.com/goplus/xai/spec"
	"google.golang.org/genai"
)

const (
	maxSeed = 4294967295
)

// validateGenVideoConfig validates GenerateVideosConfig and GenerateVideosSource
// against Veo API constraints before calling the backend.
func validateGenVideoConfig(model string, source *genai.GenerateVideosSource, config *genai.GenerateVideosConfig) error {
	if source == nil {
		return fmt.Errorf("xai: video source is required")
	}
	prompt := strings.TrimSpace(source.Prompt)
	if source.Image == nil && source.Video == nil && prompt == "" {
		return fmt.Errorf("xai: Prompt is required for text-to-video")
	}
	if config == nil {
		return nil
	}

	videoSchema := VideoSchemaFor(model)

	if config.DurationSeconds != nil {
		if err := validateIntRestriction(videoSchema, ParamDurationSeconds, int64(*config.DurationSeconds)); err != nil {
			return err
		}
	}

	if config.NumberOfVideos != 0 {
		if err := validateIntRestriction(videoSchema, ParamNumberOfVideos, int64(config.NumberOfVideos)); err != nil {
			return err
		}
	}

	// Seed: 0-4294967295
	if config.Seed != nil {
		s := int64(*config.Seed)
		if s < 0 || s > maxSeed {
			return fmt.Errorf("xai: Seed %d not in [0, %d]", *config.Seed, maxSeed)
		}
	}

	// String enum validation via restriction_genVideo
	schema := newInputSchema(&struct {
		genai.GenerateVideosSource
		genai.GenerateVideosConfig
	}{}, restriction_genVideo)

	for name, val := range map[string]string{
		"AspectRatio":        config.AspectRatio,
		"Resolution":         config.Resolution,
		"PersonGeneration":   config.PersonGeneration,
		"CompressionQuality": string(config.CompressionQuality),
	} {
		if val == "" {
			continue
		}
		if r := schema.Restrict(name); r != nil {
			if err := r.ValidateString(name, val); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateIntRestriction(schema xai.VideoSchema, name string, value int64) error {
	if schema == nil {
		return nil
	}
	if r := schema.Restrict(name); r != nil {
		if err := r.ValidateInt(name, value); err != nil {
			return err
		}
	}
	return nil
}
