/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

// Video GenVideo demo: pick provider from model id (Qiniu/Volc Seedance vs Qiniu Kling).
//
// Usage:
//
//	go run ./examples/videorouter [model_id]
//
// Examples:
//
//	go run ./examples/videorouter doubao-seedance-2-0-260128   # prefers QINIU_API_KEY, falls back to ARK_API_KEY
//	go run ./examples/videorouter kling-v2-5-turbo             # needs QINIU_API_KEY for real Kling
//
// Without API keys, each line uses the corresponding mock (placeholder video URL).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/kling"
	"github.com/goplus/xai/spec/seedance"

	"github.com/goplus/xai/examples/shared"
)

func main() {
	modelID := seedance.ModelDoubaoSeedance20
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		modelID = strings.TrimSpace(os.Args[1])
	}
	RunVideoGen(xai.Model(modelID))
}

// RunVideoGen loads the service for the model, sets provider-specific params, then Call.
func RunVideoGen(model xai.Model) {
	svc, provider, err := shared.VideoGenServiceForModel(model)
	if err != nil {
		fmt.Println("VideoGenServiceForModel:", err)
		os.Exit(1)
	}
	fmt.Printf("model=%q provider=%s\n", model, provider)

	ctx := context.Background()
	op, err := svc.Operation(model, xai.GenVideo)
	if err != nil {
		fmt.Println("Operation:", err)
		os.Exit(1)
	}

	switch {
	case seedance.IsVideoModel(string(model)):
		op.Params().(*seedance.Params).
			Set(seedance.ParamPrompt, "A short cinematic product shot, soft light.").
			Set(seedance.ParamRatio, "16:9").
			Set(seedance.ParamDuration, 5).
			Set(seedance.ParamGenerateAudio, true).
			Set(seedance.ParamWatermark, false)
	case kling.IsVideoModel(string(model)):
		// kling-v2-5-turbo allows prompt-only GenVideo
		op.Params().(*kling.Params).
			Set(kling.ParamPrompt, "一只小狗在草地上奔跑，电影感，高清").
			Set(kling.ParamSeconds, "5")
	default:
		fmt.Println("internal error: model not classified")
		os.Exit(1)
	}

	resp, err := xai.CallSync(ctx, svc, op, svc.Options())
	if err != nil {
		fmt.Println("CallSync:", err)
		os.Exit(1)
	}
	if !resp.Done() {
		if tid := resp.TaskID(); tid != "" {
			fmt.Println("submitted, task_id:", tid, "(async: polling every ~2s until done)")
		} else {
			fmt.Println("submitted, waiting for async completion…")
		}
	}

	results, err := xai.Wait(ctx, svc, resp, func(r xai.OperationResponse) {
		if !r.Done() {
			fmt.Println("polling… task_id:", r.TaskID())
		}
	})
	if err != nil {
		fmt.Println("Wait:", err)
		os.Exit(1)
	}
	fmt.Println("done.")
	for i := 0; i < results.Len(); i++ {
		v := results.At(i).(*xai.OutputVideo)
		fmt.Println("video:", v.Video.StgUri())
	}
}
