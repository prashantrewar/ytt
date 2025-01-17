// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package yttlibrary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"carvel.dev/ytt/pkg/orderedmap"
	"carvel.dev/ytt/pkg/template/core"
	"carvel.dev/ytt/pkg/yamlmeta"
	"github.com/k14s/starlark-go/starlark"
	"github.com/k14s/starlark-go/starlarkstruct"
)

var (
	// JSONAPI contains the definition of the @ytt:json module
	JSONAPI = starlark.StringDict{
		"json": &starlarkstruct.Module{
			Name: "json",
			Members: starlark.StringDict{
				"encode": starlark.NewBuiltin("json.encode", core.ErrWrapper(jsonModule{}.Encode)),
				"decode": starlark.NewBuiltin("json.decode", core.ErrWrapper(jsonModule{}.Decode)),
			},
		},
	}
)

type jsonModule struct{}

// Encode is a core.StarlarkFunc that renders the provided input into a JSON formatted string
func (b jsonModule) Encode(thread *starlark.Thread, f *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if args.Len() != 1 {
		return starlark.None, fmt.Errorf("expected exactly one argument")
	}
	allowedKWArgs := map[string]struct{}{
		"indent": {},
		"escape_html": {},
	}
	if err := core.CheckArgNames(kwargs, allowedKWArgs); err != nil {
		return starlark.None, err
	}

	val, err := core.NewStarlarkValue(args.Index(0)).AsGoValue()
	if err != nil {
		return starlark.None, err
	}
	val = orderedmap.Conversion{yamlmeta.NewGoFromAST(val)}.AsUnorderedStringMaps()

	var valBs []byte
	buffer := bytes.NewBuffer(valBs)
	indent, err := core.Int64Arg(kwargs, "indent")
	if err != nil {
		return starlark.None, err
	}
	if indent < 0 || indent > 8 {
		// mitigate https://cwe.mitre.org/data/definitions/409.html
		return starlark.None, fmt.Errorf("indent value must be between 0 and 8")
	}

	escapeHTML, err := core.BoolArg(kwargs, "escape_html", true)
	if err != nil {
		return starlark.None, err
	}

	encoder := json.NewEncoder(buffer)
	if indent > 0 {
		encoder.SetIndent("", strings.Repeat(" ", int(indent)))
	}
	encoder.SetEscapeHTML(escapeHTML)

	if err := encoder.Encode(val); err != nil {
		return starlark.None, err
	}

	res := strings.TrimSuffix(buffer.String(), "\n")
	return starlark.String(res), nil
}

// Decode is a core.StarlarkFunc that parses the provided input from JSON format into dicts, lists, and scalars
func (b jsonModule) Decode(thread *starlark.Thread, f *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if args.Len() != 1 {
		return starlark.None, fmt.Errorf("expected exactly one argument")
	}

	valEncoded, err := core.NewStarlarkValue(args.Index(0)).AsString()
	if err != nil {
		return starlark.None, err
	}

	var valDecoded interface{}

	err = json.Unmarshal([]byte(valEncoded), &valDecoded)
	if err != nil {
		return starlark.None, err
	}

	valDecoded = orderedmap.Conversion{valDecoded}.FromUnorderedMaps()

	return core.NewGoValue(valDecoded).AsStarlarkValue(), nil
}
