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

// Run: go run ./examples/kling/video kling-v2-5-turbo
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/shared"
)

// RunKlingV25Turbo runs all kling-v2-5-turbo demos (text2video, img2video, keyframe).
func RunKlingV25Turbo() {
	svc, err := shared.NewService()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ctx := context.Background()

	// 1. text2video
	fmt.Println("--- text2video ---")
	op, _ := svc.Operation(xai.Model(kling.ModelKlingV25Turbo), xai.GenVideo)
	op.Params().Set(kling.ParamPrompt, "a hero enters the battlefield, dramatic lighting, slow motion")
	op.Params().Set(kling.ParamMode, kling.ModePro)
	op.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op.Params().Set(kling.ParamNegativePrompt, "blurry, low quality")
	results, _ := xai.Call(ctx, svc, op, svc.Options(), nil)
	printVideoResults("kling-v2-5-turbo", "text2video", results)

	// 2. img2video
	fmt.Println("--- img2video ---")
	op2, _ := svc.Operation(xai.Model(kling.ModelKlingV25Turbo), xai.GenVideo)
	op2.Params().Set(kling.ParamPrompt, "人在奔跑")
	op2.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op2.Params().Set(kling.ParamSize, kling.Size1280x720)
	op2.Params().Set(kling.ParamMode, kling.ModePro)
	results2, _ := xai.Call(ctx, svc, op2, svc.Options(), nil)
	printVideoResults("kling-v2-5-turbo", "img2video", results2)

	// 3. keyframe (首尾帧生视频)
	fmt.Println("--- keyframe ---")
	op3, _ := svc.Operation(xai.Model(kling.ModelKlingV25Turbo), xai.GenVideo)
	op3.Params().Set(kling.ParamPrompt, "人在跑到了天涯海角")
	op3.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op3.Params().Set(kling.ParamImageTail, DemoVideoURLs.EndFrame)
	op3.Params().Set(kling.ParamSize, kling.Size1280x720)
	op3.Params().Set(kling.ParamMode, kling.ModePro)
	results3, _ := xai.Call(ctx, svc, op3, svc.Options(), nil)
	printVideoResults("kling-v2-5-turbo", "keyframe", results3)
}
