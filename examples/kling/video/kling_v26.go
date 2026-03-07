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

// Run: go run ./examples/kling/video kling-v2-6
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/shared"
)

// RunKlingV26 runs all kling-v2-6 demos (text2video, img2video, motion_control, sound_video).
func RunKlingV26() {
	svc, err := shared.NewService()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ctx := context.Background()

	// 1. text2video
	fmt.Println("--- text2video ---")
	op, _ := svc.Operation(xai.Model(kling.ModelKlingV26), xai.GenVideo)
	op.Params().Set(kling.ParamPrompt, "一只可爱的橘猫在阳光下奔跑，慢镜头，电影质感")
	op.Params().Set(kling.ParamMode, kling.ModePro)
	op.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op.Params().Set(kling.ParamSize, kling.Size1920x1080)
	results, err := xai.Call(ctx, svc, op, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-6", "text2video", results)

	// 2. img2video
	fmt.Println("--- img2video ---")
	op2, _ := svc.Operation(xai.Model(kling.ModelKlingV26), xai.GenVideo)
	op2.Params().Set(kling.ParamPrompt, "让图片中的角色动起来，镜头缓慢右移")
	op2.Params().Set(kling.ParamInputReference, DemoVideoURLs.FirstFrame)
	op2.Params().Set(kling.ParamMode, kling.ModePro)
	op2.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op2.Params().Set(kling.ParamSize, kling.Size1920x1080)
	results2, err := xai.Call(ctx, svc, op2, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-6", "img2video", results2)

	// 3. motion_control (动作控制)
	fmt.Println("--- motion_control ---")
	op3, _ := svc.Operation(xai.Model(kling.ModelKlingV26), xai.GenVideo)
	op3.Params().Set(kling.ParamPrompt, "女孩穿着灰色宽松T恤和牛仔短裤")
	op3.Params().Set(kling.ParamImageUrl, DemoVideoURLs.MotionImage)
	op3.Params().Set(kling.ParamVideoUrl, DemoVideoURLs.MotionVideo)
	op3.Params().Set(kling.ParamCharacterOrientation, "image")
	op3.Params().Set(kling.ParamKeepOriginalSound, "yes")
	op3.Params().Set(kling.ParamMode, kling.ModePro)
	results3, err := xai.Call(ctx, svc, op3, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-6", "motion_control", results3)

	// 4. sound_video (有声视频)
	fmt.Println("--- sound_video ---")
	op4, _ := svc.Operation(xai.Model(kling.ModelKlingV26), xai.GenVideo)
	op4.Params().Set(kling.ParamPrompt, "一个人在演讲")
	op4.Params().Set(kling.ParamMode, kling.ModePro)
	op4.Params().Set(kling.ParamSound, kling.SoundOn)
	op4.Params().Set(kling.ParamSeconds, kling.Seconds5)
	results4, err := xai.Call(ctx, svc, op4, svc.Options(), nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printVideoResults("kling-v2-6", "sound_video", results4)
}
