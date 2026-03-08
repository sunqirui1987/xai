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

package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"google.golang.org/genai"

	"github.com/goplus/xai/examples/gemini/shared"
)

func runImageEdit() {
	svc := shared.NewService("")
	ctx := context.Background()

	op, err := svc.Operation(xai.Model(shared.ModelFlashImage), xai.EditImage)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	img1 := svc.ImageFromStgUri(xai.ImageJPEG, DemoURLs.RunningMan)
	img2 := svc.ImageFromStgUri(xai.ImageJPEG, DemoURLs.Lawn)
	ref1, _ := svc.ReferenceImage(img1, 0, xai.RawReferenceImage)
	ref2, _ := svc.ReferenceImage(img2, 1, xai.StyleReferenceImage)

	op.Params().
		Set("Prompt", "结合这两张图片的风格，生成一张新的艺术作品").
		Set("References", []genai.ReferenceImage{ref1.(genai.ReferenceImage), ref2.(genai.ReferenceImage)}).
		Set("AspectRatio", "16:9")

	resp, err := op.Call(ctx, svc, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printImageResults(resp.Results())
}
