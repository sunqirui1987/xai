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

// Run: go run ./examples/kling/video kling-v2-1
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/shared"
)

// RunKlingV21 runs all kling-v2-1 demos (img2video, keyframe). V2.1 不支持纯文生视频。
func RunKlingV21() {
	svc, err := shared.NewService()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ctx := context.Background()

	// 1. img2video
	fmt.Println("--- img2video ---")
	op, _ := svc.Operation(xai.Model(kling.ModelKlingV21Video), xai.GenVideo)
	op.Params().Set(kling.ParamPrompt, "camera pans slowly to the right, cinematic")
	op.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op.Params().Set(kling.ParamMode, kling.ModePro)
	op.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op.Params().Set(kling.ParamSize, kling.Size1920x1080)
	results, err := xai.Call(ctx, svc, op, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-1", "img2video", results)

	// 2. keyframe (首尾帧生视频)
	fmt.Println("--- keyframe ---")
	op2, _ := svc.Operation(xai.Model(kling.ModelKlingV21Video), xai.GenVideo)
	op2.Params().Set(kling.ParamPrompt, "smooth transition from day to night")
	op2.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op2.Params().Set(kling.ParamImageTail, DemoVideoURLs.EndFrame)
	op2.Params().Set(kling.ParamMode, kling.ModePro)
	op2.Params().Set(kling.ParamSeconds, kling.Seconds10)
	op2.Params().Set(kling.ParamNegativePrompt, "jittery, unstable")
	results2, err := xai.Call(ctx, svc, op2, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-1", "keyframe", results2)
}
