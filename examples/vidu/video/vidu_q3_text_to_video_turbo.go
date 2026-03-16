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

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/vidu"
)

func runViduQ3TextToVideoTurbo(ctx context.Context, svc xai.Service, model xai.Model) error {
	op, err := svc.Operation(model, xai.GenVideo)
	if err != nil {
		return err
	}
	op.Params().
		Set(vidu.ParamPrompt, "一只可爱的橘猫在阳光下追逐蝴蝶，慢镜头，电影质感，温暖的光线").
		Set(vidu.ParamSeed, 1).
		Set(vidu.ParamDuration, 5).
		Set(vidu.ParamResolution, vidu.Resolution1080p).
		Set(vidu.ParamMovementAmplitude, "auto")

	results, err := xai.Call(ctx, svc, op, newViduOptions(svc, "demo-user-q3-text-turbo"), progressPrinter("q3-text-turbo"))
	if err != nil {
		return err
	}
	printVideoResults(string(model), "text-to-video-turbo", results)
	return nil
}
