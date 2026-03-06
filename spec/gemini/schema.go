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
	"reflect"

	xai "github.com/goplus/xai/spec"
)

// -----------------------------------------------------------------------------

type opParams struct {
	v reflect.Value
}

func newParams(params any) *opParams {
	return &opParams{v: reflect.ValueOf(params)}
}

func (p *opParams) Set(name string, val any) xai.Params {
	panic("todo")
}

// -----------------------------------------------------------------------------

type opResults struct {
	v reflect.Value
}

func newResults(results any) *opResults {
	return &opResults{v: reflect.ValueOf(results)}
}

func (p *opResults) Get(name string) any {
	panic("todo")
}

// -----------------------------------------------------------------------------
