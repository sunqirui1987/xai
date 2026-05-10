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

// APIMart GPT-Image-2 examples.
// Run: go run ./examples/apimart [demo]
// Set APIMART_API_KEY for real API calls.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	xai "github.com/goplus/xai/spec"
	apimart "github.com/goplus/xai/spec/openai/provider/apimart"
)

var demos = map[string]func(){
	"generate": runGenerate,
	"edit":     runEdit,
	"resume":   runResume,
}

var demoOrder = []string{"generate", "edit", "resume"}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("APIMart GPT-Image-2 examples:")
		fmt.Println("  Set APIMART_API_KEY for real API calls")
		fmt.Println()
		for _, name := range demoOrder {
			fmt.Printf("  %-10s %s\n", name, demoDesc(name))
		}
		fmt.Println()
		fmt.Println("Usage: go run ./examples/apimart [demo]")
		return
	}

	for _, arg := range args {
		if fn, ok := demos[arg]; ok {
			fmt.Println("---", arg, "---")
			fn()
		} else {
			fmt.Printf("Unknown demo: %s\nAvailable: %v\n", arg, demoOrder)
		}
	}
}

func demoDesc(name string) string {
	switch name {
	case "generate":
		return "Text-to-image with size=16:9 resolution=2k"
	case "edit":
		return "Image-to-image with reference image_urls"
	case "resume":
		return "Resume polling with APIMART_TASK_ID"
	default:
		return ""
	}
}

func newService() *apimart.Service {
	apiKey := strings.TrimSpace(os.Getenv("APIMART_API_KEY"))
	return apimart.NewService(apiKey)
}

func runGenerate() {
	svc := newService()
	op, err := svc.Operation(apimart.ModelGPTImage2, xai.GenImage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	op.Params().
		Set("Prompt", "一只橘猫坐在窗台上看夕阳，水彩画风格").
		Set("Size", "16:9").
		Set("Resolution", "2k")

	runOperation(context.Background(), svc, apimart.ModelGPTImage2, xai.GenImage, op)
}

func runEdit() {
	svc := newService()
	op, err := svc.Operation(apimart.ModelGPTImage2, xai.EditImage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	op.Params().
		Set("Prompt", "把这张照片变成水彩插画风格，保留主体构图").
		Set("Size", "4:3").
		Set("Resolution", "2k").
		Set("OfficialFallback", true).
		Set("Images", []string{
			"https://images.unsplash.com/photo-1543852786-1cf6624b9987?auto=format&fit=crop&w=1200&q=80",
		})

	runOperation(context.Background(), svc, apimart.ModelGPTImage2, xai.EditImage, op)
}

func runResume() {
	svc := newService()
	taskID := strings.TrimSpace(os.Getenv("APIMART_TASK_ID"))
	if taskID == "" {
		fmt.Println("Set APIMART_TASK_ID to resume polling an existing task")
		return
	}

	ctx := context.Background()
	resp, err := xai.GetTask(ctx, svc, apimart.ModelGPTImage2, xai.GenImage, taskID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	waitAndPrint(ctx, svc, resp)
}

func runOperation(ctx context.Context, svc *apimart.Service, model xai.Model, action xai.Action, op xai.Operation) {
	resp, err := xai.CallSync(ctx, svc, op, svc.Options())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if taskID := resp.TaskID(); taskID != "" {
		fmt.Println("task_id:", taskID)

		// Simulate loading task_id from storage and restoring the async response.
		resp, err = xai.GetTask(ctx, svc, model, action, taskID)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}
	waitAndPrint(ctx, svc, resp)
}

func waitAndPrint(ctx context.Context, svc *apimart.Service, resp xai.OperationResponse) {
	results, err := xai.Wait(ctx, svc, resp, func(resp xai.OperationResponse) {
		if taskID := resp.TaskID(); taskID != "" {
			fmt.Println("polling task:", taskID)
		}
	})
	if err != nil {
		fmt.Println("Wait error:", err)
		return
	}
	printImageResults(results)
}

func printImageResults(results xai.Results) {
	fmt.Printf("results { images: %d }\n", results.Len())
	for i := 0; i < results.Len(); i++ {
		out := results.At(i).(*xai.OutputImage)
		fmt.Printf("  image[%d]: %s (%s)\n", i, out.URL(), out.Image.Type())
	}
}
