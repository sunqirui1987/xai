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

package schema

import (
	"log"
	"reflect"

	xai "github.com/goplus/xai/spec"
)

// -----------------------------------------------------------------------------

type paramsAdapter interface {
	ToUnderlying(val any) any
	SetBasicOpt(dst, val reflect.Value, vkind reflect.Kind) bool
}

type Params[T paramsAdapter] struct {
	v reflect.Value
}

func NewParams[T paramsAdapter](params any) *Params[T] {
	return &Params[T]{v: reflect.ValueOf(params).Elem()}
}

func (p *Params[T]) Set(name string, val any) xai.Params {
	fld := p.v.FieldByName(name)
	if fld.CanSet() {
		if val == nil {
			fld.SetZero()
			return p
		}

		// convert a spec-layer object to underlying-layer
		var adapter T
		val = adapter.ToUnderlying(val)

		v := reflect.ValueOf(val)
		vkind := v.Kind()
		if vkind >= reflect.Bool && vkind <= reflect.Float64 || vkind == reflect.String {
			if !adapter.SetBasicOpt(fld, v, vkind) {
				SetBasic(fld, v, vkind)
			}
		} else {
			fld.Set(v)
		}
	} else {
		log.Println("cannot set field:", name)
	}
	return p
}

func SetBasic(fld, v reflect.Value, vkind reflect.Kind) {
	if vkind >= reflect.Int && vkind <= reflect.Int64 {
		if kind := fld.Kind(); kind >= reflect.Int && kind <= reflect.Int64 {
			fld.SetInt(v.Int())
		} else {
			fld.SetFloat(float64(v.Int()))
		}
		return
	}
	switch vkind {
	case reflect.String:
		fld.SetString(v.String())
	case reflect.Bool:
		fld.SetBool(v.Bool())
	case reflect.Float32, reflect.Float64:
		fld.SetFloat(v.Float())
	default:
		fld.Set(v)
	}
}

// -----------------------------------------------------------------------------

type PointerAsOpt struct{}

func (PointerAsOpt) SetBasicOpt(fld, v reflect.Value, vkind reflect.Kind) (ok bool) {
	ok = fld.Kind() == reflect.Pointer
	if ok {
		pv := reflect.New(fld.Type().Elem())
		SetBasic(pv.Elem(), v, vkind)
		fld.Set(pv)
	}
	return
}

// -----------------------------------------------------------------------------
