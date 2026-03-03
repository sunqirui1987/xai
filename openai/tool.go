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

package openai

import (
	"encoding/json"
	"strings"
	"unsafe"

	"github.com/goplus/xai"
	"github.com/openai/openai-go/v3/responses"
)

// -----------------------------------------------------------------------------

func (p *contentBuilder) ToolUse(toolID, name string, input any) xai.ContentBuilder {
	var (
		content responses.ResponseInputItemUnionParam
	)
	if strings.HasPrefix(name, "std/") {
		panic("todo")
	} else {
		var args []byte
		var err error
		if v, ok := input.(json.RawMessage); ok {
			args = []byte(v)
		} else {
			args, err = json.Marshal(input)
			if err != nil {
				panic("invalid tool input: " + err.Error())
			}
		}
		content = responses.ResponseInputItemParamOfFunctionCall(toolID, unsafe.String(unsafe.SliceData(args), len(args)), name)
	}
	return p.addNonMsg(content)
}

func (p *contentBuilder) ToolResult(toolID, name string, result any, isError bool) xai.ContentBuilder {
	// TODO(xsw): validate content
	panic("todo")
}

// -----------------------------------------------------------------------------
