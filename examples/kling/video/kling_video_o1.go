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

// Run: go run ./examples/kling/video kling-video-o1
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"

	"github.com/goplus/xai/examples/shared"
)

// RunKlingVideoO1 runs all kling-video-o1 demos (text2video, img2video, keyframe, multi_ref, video2video).
func RunKlingVideoO1() {
	svc, err := shared.NewService()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ctx := context.Background()

	// 1. text2video
	fmt.Println("--- text2video ---")
	op, _ := svc.Operation(xai.Model(kling.ModelKlingVideoO1), xai.GenVideo)
	op.Params().Set(kling.ParamPrompt, "一只可爱的橘猫在阳光下奔跑，慢镜头，电影质感")
	op.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op.Params().Set(kling.ParamMode, kling.ModePro)
	op.Params().Set(kling.ParamSeconds, kling.Seconds5)
	results, _ := xai.Call(ctx, svc, op, svc.Options(), nil)
	printVideoResults("kling-video-o1", "text2video", results)

	// 2. img2video
	fmt.Println("--- img2video ---")
	op2, _ := svc.Operation(xai.Model(kling.ModelKlingVideoO1), xai.GenVideo)
	op2.Params().Set(kling.ParamPrompt, "这个人在跑马拉松")
	op2.Params().Set(kling.ParamImageList, []map[string]interface{}{
		{"image": DemoVideoURLs.FirstFrame},
	})
	op2.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op2.Params().Set(kling.ParamMode, kling.ModePro)
	results2, _ := xai.Call(ctx, svc, op2, svc.Options(), nil)
	printVideoResults("kling-video-o1", "img2video", results2)

	// 3. keyframe (首尾帧生视频)
	fmt.Println("--- keyframe ---")
	op3, _ := svc.Operation(xai.Model(kling.ModelKlingVideoO1), xai.GenVideo)
	op3.Params().Set(kling.ParamPrompt, "视频连贯在一起")
	op3.Params().Set(kling.ParamImageList, []map[string]interface{}{
		{"image": DemoVideoURLs.MultiRef1, "type": "first_frame"},
		{"image": DemoVideoURLs.MultiRef2, "type": "end_frame"},
	})
	op3.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op3.Params().Set(kling.ParamMode, kling.ModePro)
	results3, _ := xai.Call(ctx, svc, op3, svc.Options(), nil)
	printVideoResults("kling-video-o1", "keyframe", results3)

	// 4. multi_ref (多图参考)
	fmt.Println("--- multi_ref ---")
	op4, _ := svc.Operation(xai.Model(kling.ModelKlingVideoO1), xai.GenVideo)
	op4.Params().Set(kling.ParamImageList, []map[string]interface{}{
		{"image": DemoVideoURLs.MultiRef1},
		{"image": DemoVideoURLs.MultiRef2, "type": "first_frame"},
	})
	op4.Params().Set(kling.ParamPrompt, "blend the styles and create a cohesive scene")
	op4.Params().Set(kling.ParamSeconds, kling.Seconds5)
	op4.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op4.Params().Set(kling.ParamMode, kling.ModePro)
	op4.Params().Set(kling.ParamVideoMode, "multi_ref")
	results4, _ := xai.Call(ctx, svc, op4, svc.Options(), nil)
	printVideoResults("kling-video-o1", "multi_ref", results4)

	// 5. video2video (refer_type: feature)
	fmt.Println("--- video2video ---")
	op5, _ := svc.Operation(xai.Model(kling.ModelKlingVideoO1), xai.GenVideo)
	op5.Params().Set(kling.ParamPrompt, "画面中增加一个小狗")
	op5.Params().Set(kling.ParamVideoList, []map[string]interface{}{
		{"video_url": DemoVideoURLs.VideoFeature, "refer_type": "feature", "keep_original_sound": "yes"},
	})
	op5.Params().Set(kling.ParamSize, kling.Size1920x1080)
	op5.Params().Set(kling.ParamMode, kling.ModePro)
	results5, _ := xai.Call(ctx, svc, op5, svc.Options(), nil)
	printVideoResults("kling-video-o1", "video2video", results5)
}
