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
 * See the License for the specific language governing permissions and limitations under the License.
 */

// Run: go run ./examples/kling/images
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/kling/output"
)

// DemoImageURLs holds real, publicly accessible image URLs for Kling image examples.
var DemoImageURLs = struct {
	RefStyle   string // for image2image reference_images
	Subject1   string // for multi_image
	Subject2   string // for multi_image
	RunningMan string // for image2image, single image edit
}{
	RefStyle:   "https://huggingface.co/datasets/huggingface/documentation-images/resolve/4a5c8349eb8172fff604d547dc4991fbab6078e3/diffusers/controlnet-img2img.jpg",
	Subject1:   "https://aitoken-public.qnaigc.com/example/generate-image/smile-woman.png",
	Subject2:   "https://aitoken-public.qnaigc.com/example/generate-image/image-to-image-with-mask-1.jpg",
	RunningMan: "https://aitoken-public.qnaigc.com/example/generate-video/running-man.jpg",
}

// printImageResults prints and saves image URLs and curl command for verification.
// params should be from op.Params().(*kling.Params).Export(); if nil, no curl is built.
func printImageResults(model, demo string, results xai.Results, params map[string]any) {
	urls := make([]string, 0, results.Len())
	for i := 0; i < results.Len(); i++ {
		out := results.At(i).(*xai.OutputImage)
		url := out.Image.StgUri()
		urls = append(urls, url)
		fmt.Println("  Image URL:", url)
	}

	curl := buildCurl(model, params)
	if curl != "" {
		fmt.Println("  Curl:")
		fmt.Println(curl)
	}
	output.Append(model, demo, urls, curl)
}

// buildCurl builds a curl command for the given model and params.
func buildCurl(model string, params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	baseURL := "https://api.qnaigc.com"
	token := os.Getenv("QINIU_API_KEY")
	if token == "" {
		token = "<token>"
	}

	m := strings.ToLower(model)
	var url, body string

	switch m {
	case kling.ModelKlingImageO1:
		// POST /queue/fal-ai/kling-image/o1
		url = baseURL + "/queue/fal-ai/kling-image/o1"
		bodyMap := map[string]any{
			"prompt":       params[kling.ParamPrompt],
			"resolution":   params[kling.ParamResolution],
			"aspect_ratio": params[kling.ParamAspectRatio],
		}
		if n := getInt(params[kling.ParamN]); n > 0 {
			bodyMap["num_images"] = n
		}
		if refs, ok := params[kling.ParamReferenceImages].([]string); ok && len(refs) > 0 {
			bodyMap["image_urls"] = refs
		}
		b, _ := json.MarshalIndent(bodyMap, "", "    ")
		body = string(b)
	default:
		// kling-v1, v1-5, v2, v2-new, v2-1: /v1/images/generations or /v1/images/edits
		subjectList, _ := params[kling.ParamSubjectImageList]
		var useEdits bool
		if list, ok := subjectList.([]map[string]string); ok && len(list) >= 2 && len(list) <= 4 {
			useEdits = true
		}
		if list, ok := subjectList.([]interface{}); ok && len(list) >= 2 && len(list) <= 4 {
			useEdits = true
		}

		if useEdits {
			url = baseURL + "/v1/images/edits"
		} else {
			url = baseURL + "/v1/images/generations"
		}

		bodyMap := make(map[string]any)
		for k, v := range params {
			if strings.HasPrefix(k, "_") {
				continue
			}
			bodyMap[k] = v
		}
		bodyMap["model"] = model
		if useEdits && bodyMap["image"] == nil {
			bodyMap["image"] = ""
		}
		b, _ := json.MarshalIndent(bodyMap, "", "    ")
		body = string(b)
	}

	return fmt.Sprintf("curl --location --request POST '%s' \\\n--header 'Authorization: Bearer %s' \\\n--header 'Content-Type: application/json' \\\n--data-raw '%s'",
		url, token, strings.ReplaceAll(body, "'", "'\\''"))
}

func getInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}
