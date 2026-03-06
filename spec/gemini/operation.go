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

package gemini

import (
	"context"
	"time"

	xai "github.com/goplus/xai/spec"
	"google.golang.org/genai"
)

// -----------------------------------------------------------------------------

func (p *Service) Actions(model xai.Model) []xai.Action {
	return []xai.Action{
		xai.GenVideo,
		xai.GenImage,
		xai.EditImage,
		xai.RecontextImage,
		xai.SegmentImage,
		xai.UpscaleImage,
	}
}

func (p *Service) Operation(model xai.Model, action xai.Action) (op xai.Operation, err error) {
	switch action {
	case xai.GenVideo:
		op = &genVideo{}
	default:
		err = xai.ErrNotFound
	}
	return
}

// -----------------------------------------------------------------------------

type genVideoResp struct {
	op *genai.GenerateVideosOperation
}

func (p genVideoResp) Done() bool {
	return p.op.Done
}

func (p genVideoResp) Sleep() {
	time.Sleep(15 * time.Second)
}

func (p genVideoResp) Retry(ctx context.Context, svc xai.Service) (xai.OperationResponse, error) {
	op, err := svc.(*Service).ops.GetVideosOperation(ctx, p.op, nil)
	if err != nil {
		return nil, err
	}
	return genVideoResp{op}, nil
}

func (p genVideoResp) Results() xai.Results {
	return newResults(p.op.Response)
}

type genVideo struct {
	genai.GenerateVideosConfig
	Image *genai.Image

	model string
}

func (p *genVideo) InputSchema() xai.Schema {
	panic("todo")
}

func (p *genVideo) Params() xai.Params {
	return newParams(p)
}

func (p *genVideo) Call(ctx context.Context, svc xai.Service, prompt string, opts xai.OptionBuilder) (resp xai.OperationResponse, err error) {
	if v, ok := opts.(*options); ok {
		p.HTTPOptions = &v.opts
	}
	op, err := svc.(*Service).models.GenerateVideos(ctx, p.model, prompt, p.Image, &p.GenerateVideosConfig)
	if err != nil {
		return
	}
	return genVideoResp{op}, nil
}

// -----------------------------------------------------------------------------
