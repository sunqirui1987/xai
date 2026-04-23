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

// go run ./examples/openai gptimage
// go run ./examples/openai gptimage-edit

package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"

	"github.com/goplus/xai/examples/openai/shared"
)

func runGPTImageGenerate() {
	svc := shared.NewService("")
	ctx := context.Background()

	op, err := svc.Operation(xai.Model("openai/gpt-image-2"), xai.GenImage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	op.Params().
		Set("Prompt", "可爱的少女，动漫").
		Set("Quality", "high")

	resp, err := op.Call(ctx, svc, shared.DebugOptions(svc, nil))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printImageResults(resp.Results())
}

func runGPTImageEdit() {
	svc := shared.NewService("")
	ctx := context.Background()

	op, err := svc.Operation(xai.Model("openai/gpt-image-2"), xai.EditImage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	op.Params().
		Set("Prompt", "图片中增加一个人").
		Set("Quality", "low").
		Set("Images", []string{DemoURLs.RunningManImage})

	resp, err := op.Call(ctx, svc, shared.DebugOptions(svc, nil))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printImageResults(resp.Results())
}

func printImageResults(results xai.Results) {
	fmt.Printf("images { count: %d }\n", results.Len())
	for i := 0; i < results.Len(); i++ {
		item := results.At(i).(*xai.OutputImage)
		fmt.Printf("  image[%d] { type: %q, url: %q }\n", i, item.Image.Type(), item.URL())
	}
}
