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

// Run: go run ./examples/kling/video kling-v3
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/shared"
)

// RunKlingV3 runs all kling-v3 demos (text2video, img2video, sound_video).
func RunKlingV3() {
	svc, err := shared.NewService()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ctx := context.Background()

	// 1. text2video
	fmt.Println("--- text2video ---")
	op, _ := svc.Operation(xai.Model(kling.ModelKlingV3), xai.GenVideo)
	op.Params().Set(kling.ParamPrompt, "一只可爱的小猫在阳光下玩耍")
	op.Params().Set(kling.ParamMode, kling.ModeStd)
	op.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op.Params().Set(kling.ParamSize, kling.Size1280x720)
	results, _ := xai.Call(ctx, svc, op, svc.Options(), nil)
	printVideoResults("kling-v3", "text2video", results)

	// 2. img2video
	fmt.Println("--- img2video ---")
	op2, _ := svc.Operation(xai.Model(kling.ModelKlingV3), xai.GenVideo)
	op2.Params().Set(kling.ParamPrompt, "让图片中的角色动起来")
	op2.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op2.Params().Set(kling.ParamMode, kling.ModePro)
	op2.Params().Set(kling.ParamSeconds, kling.Seconds5)
	results2, _ := xai.Call(ctx, svc, op2, svc.Options(), nil)
	printVideoResults("kling-v3", "img2video", results2)

	// 3. sound_video (有声视频)
	fmt.Println("--- sound_video ---")
	op3, _ := svc.Operation(xai.Model(kling.ModelKlingV3), xai.GenVideo)
	op3.Params().Set(kling.ParamPrompt, "一个人在演讲")
	op3.Params().Set(kling.ParamMode, kling.ModePro)
	op3.Params().Set(kling.ParamSound, kling.SoundOn)
	op3.Params().Set(kling.ParamSeconds, kling.Seconds5)
	results3, _ := xai.Call(ctx, svc, op3, svc.Options(), nil)
	printVideoResults("kling-v3", "sound_video", results3)
}
