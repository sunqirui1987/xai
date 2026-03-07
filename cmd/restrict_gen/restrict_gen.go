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
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// -----------------------------------------------------------------------------

var geminiRewriteFlds = map[string]string{
	"HTTPOptions":     "",
	"SDKHTTPResponse": "",
	"Labels":          "",
	"OutputGCSURI":    "OutputStgUri",
}

var pkgRewriteFlds = map[string]map[string]string{
	"github.com/goplus/xai/spec/gemini": geminiRewriteFlds,
}

// -----------------------------------------------------------------------------

func gen(t *types.Named, rewriteFlds map[string]string) {
	echo("==>", t.Obj().Name())
	collectFields(t, rewriteFlds)
}

func collectFields(t types.Type, rewriteFlds map[string]string) {
	if struc, ok := t.Underlying().(*types.Struct); ok {
		for i, n := 0, struc.NumFields(); i < n; i++ {
			field := struc.Field(i)
			if field.Embedded() {
				collectFields(field.Type(), rewriteFlds)
			} else if field.Exported() {
				name := field.Name()
				if newName, ok := rewriteFlds[name]; ok {
					if newName == "" {
						continue
					}
					name = newName
				}
				typ := field.Type()
				if false && skipType(typ) {
					continue
				}
				echo(" ", name, typ)
			}
		}
	}
}

func skipType(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	_, ok := t.(*types.Basic)
	return ok
}

func main() {
	fset := token.NewFileSet()
	conf := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Fset: fset,
	}
	pkgs, _ := packages.Load(conf, ".")
	for _, pkg := range pkgs {
		rewriteFlds := pkgRewriteFlds[pkg.PkgPath]
		echo("package", pkg.PkgPath, rewriteFlds)
		scope := pkg.Types.Scope()
		names := scope.Names()
		for _, name := range names {
			o := scope.Lookup(name)
			if t, ok := o.Type().(*types.Named); ok {
				for i, n := 0, t.NumMethods(); i < n; i++ {
					mthd := t.Method(i)
					switch mthd.Name() {
					case "InputSchema":
						gen(t, rewriteFlds)
					}
				}
			}
		}
	}
}

func echo(v ...any) {
	fmt.Println(v...)
}

// -----------------------------------------------------------------------------
