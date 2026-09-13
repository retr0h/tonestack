// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package tools

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// handlers holds what every tool shares.
type handlers struct {
	client Client
	// device is taken by any tool that reaches the pedal, so two calls
	// never claim the editor interface at once.
	device chan struct{}
}

// Register adds tonestack's tools to a server.
//
// The tools that write to a pedal are added only when allowWrites is true. A
// device has no undo, and whoever starts the server decides.
func Register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
) {
	h := &handlers{client: c, device: make(chan struct{}, 1)}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_search",
		Description:  "Find blocks the device models, by name, real-world gear, category or instrument. Use this before naming any model: a model it does not find does not exist.",
		Annotations:  readOnly(),
		OutputSchema: outputSchema[sdk.Blocks](),
	}, h.catalogSearch)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_block",
		Description:  "One block's parameters, their ranges and defaults, and its DSP cost.",
		Annotations:  readOnly(),
		OutputSchema: outputSchema[catalog.Block](),
	}, h.catalogBlock)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_model",
		Description:  "How players set one model across measured presets: median and quartiles per parameter. A narrow spread is consensus; a wide one is taste.",
		Annotations:  readOnly(),
		OutputSchema: outputSchema[Model](),
	}, h.corpusModel)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "rigs_list",
		Description: "The rigs that ship with tonestack.",
		Annotations: readOnly(),
	}, h.rigsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "rig_show",
		Description: "One shipped rig, and the rigs that extend it.",
		Annotations: readOnly(),
	}, h.rigShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_build",
		Description: "Build a .hlx from a shipped rig or a rig file. Read what it added and what each character word moved before putting it on a pedal.",
		Annotations: &gomcp.ToolAnnotations{OpenWorldHint: new(false)},
	}, h.presetBuild)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "devices_list",
		Description: "The Line 6 Helix hardware attached over USB. HX Edit must be quit for any tool that reaches the pedal.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, h.devicesList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "presets_list",
		Description: "Every slot on the attached pedal and what it holds.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, h.presetsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_show",
		Description: "One slot on the pedal, read back as a rig.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, h.presetShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_export",
		Description: "Write one slot to a file: a rig by default, or the device's own .hlx with as=hlx.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, h.presetExport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_select",
		Description: "Load a slot on the pedal, as pressing its footswitch does. Changes nothing stored.",
		Annotations: &gomcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true},
	}, h.presetSelect)

	if !allowWrites {
		return
	}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "preset_import",
		Description: "Put a .hlx into a slot on the pedal. Whatever the slot held is saved to a file first and then gone from the pedal.",
		Annotations: destructive(),
	}, h.presetImport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "presets_copy",
		Description: "Copy one slot onto another. The destination's old preset is saved to a file first.",
		Annotations: destructive(),
	}, h.presetsCopy)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "presets_swap",
		Description: "Exchange two slots. Both are saved to files first.",
		Annotations: destructive(),
	}, h.presetsSwap)
}

// outputSchema infers T's output schema, treating catalog.ParamValue as an
// unconstrained value.
//
// A ParamValue marshals to a bare JSON literal — a number, a string or a
// bool, depending on the parameter's kind — but it holds that kind in
// unexported fields, so reflection alone describes it as an empty object.
// Left to the default inference, a real Default value then fails the SDK's
// own output validation on the very first call that carries one. The
// override says what marshalling already knows: this field's shape depends
// on data the schema cannot see.
func outputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[catalog.ParamValue](): {},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("outputSchema[%T]: %v", *new(T), err))
	}

	return s
}

// readOnly marks a tool that changes nothing anywhere.
func readOnly() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)}
}

// destructive marks a tool that overwrites what a pedal holds.
func destructive() *gomcp.ToolAnnotations {
	return &gomcp.ToolAnnotations{DestructiveHint: new(true)}
}

// said is the one line of text beside a tool's structured answer, for an agent
// that reads text rather than structure.
func said(
	format string,
	args ...any,
) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}
