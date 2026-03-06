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
	"log"
	"reflect"

	xai "github.com/goplus/xai/spec"
)

// -----------------------------------------------------------------------------

type opInputSchema struct {
	t reflect.Type
}

func (p *opInputSchema) Fields() []xai.Field {
	t := p.t
	n := t.NumField()
	fields := make([]xai.Field, 0, n)
	return getFields(fields, t, n)
}

var rewriteFlds = map[string]string{
	"HTTPOptions":  "",
	"Labels":       "",
	"OutputGCSURI": "OutputStgUri",
}

func getFields(fields []xai.Field, t reflect.Type, n int) []xai.Field {
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if f.Anonymous {
			ft := f.Type
			fields = getFields(fields, ft, ft.NumField())
		} else if f.IsExported() {
			name := f.Name
			if newName, ok := rewriteFlds[name]; ok {
				if newName == "" {
					continue
				}
				name = newName
			}
			fields = append(fields, xai.Field{
				Name: name,
				Kind: kindOf(f.Type),
			})
		}
	}
	return fields
}

var allowTypes = map[string]xai.Kind{
	"Image":                         xai.Image,
	"ProductImage":                  xai.Image,
	"ScribbleImage":                 xai.Image,
	"ReferenceImage":                xai.ReferenceImage,
	"VideoGenerationReferenceImage": xai.GenVideoReferenceImage,
	"VideoGenerationMask":           xai.GenVideoMask,
}

func kindOf(t reflect.Type) xai.Kind {
	kind := t.Kind()
	if kind == reflect.Pointer {
		t = t.Elem()
		kind = t.Kind()
	}
	switch kind {
	case reflect.Int32, reflect.Int64:
		return xai.Int
	case reflect.Float32, reflect.Float64:
		return xai.Float
	case reflect.String:
		return xai.String
	case reflect.Bool:
		return xai.Bool
	case reflect.Slice:
		return kindOf(t.Elem()) | xai.List
	case reflect.Struct, reflect.Interface:
		name := t.Name()
		if k, ok := allowTypes[name]; ok {
			return k
		}
		fallthrough
	default:
		log.Panicln("unknown field type:", t)
		return xai.Invalid
	}
}

func newInputSchema(params any) xai.InputSchema {
	return &opInputSchema{t: reflect.TypeOf(params).Elem()}
}

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
