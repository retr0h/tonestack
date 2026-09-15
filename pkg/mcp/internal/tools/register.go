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
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/retr0h/tonestack/pkg/sdk"
	"github.com/retr0h/tonestack/pkg/sdk/catalog"
)

// handlers holds what every tool shares.
type handlers struct {
	client Client
	// pedal holds a Session across the tools that reach the pedal, and
	// keeps two calls from claiming the editor interface at once.
	pedal *pedal
	// allowWrites is whether the server was started with --allow-writes. It
	// decides which tools are offered, and whether a file already on disk
	// may be written over.
	allowWrites bool
}

// Register adds tonestack's tools to a server.
//
// The tools that write to a pedal are added only when allowWrites is true, and
// without it no tool writes over a file already on disk. A device has no undo,
// nor does a file, and whoever starts the server decides.
//
// It returns what holds the pedal between device calls. Closing it lets the
// pedal go, and whoever runs the server closes it when the server stops.
func Register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
) io.Closer {
	return register(s, c, allowWrites, idleClose)
}

// register is Register, with how long the pedal stays held after a device
// call.
func register(
	s *gomcp.Server,
	c Client,
	allowWrites bool,
	idle time.Duration,
) io.Closer {
	h := &handlers{client: c, pedal: newPedal(c, idle), allowWrites: allowWrites}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_search",
		Description:  "Find blocks the device models, by name, real-world gear, category or instrument. Use this before naming any model: a model it does not find does not exist.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Blocks](),
	}, h.catalogSearch)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "catalog_block",
		Description:  "One block's parameters, their ranges and defaults, and its DSP cost.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[catalog.Block](),
	}, h.catalogBlock)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "corpus_model",
		Description:  "How players set one model across measured presets: median and quartiles per parameter. A narrow spread is consensus; a wide one is taste.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[Model](),
	}, h.corpusModel)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rigs_list",
		Description:  "The rigs that ship with tonestack.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Recipes](),
	}, h.rigsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "rig_show",
		Description:  "One shipped rig, and the rigs that extend it.",
		Annotations:  readOnly(),
		OutputSchema: mustOutputSchema[sdk.Recipe](),
	}, h.rigShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "preset_build",
		Description:  "Build a .hlx from a shipped rig or a rig file. Read what it added and what each character word moved before putting it on a pedal. Refuses a file already at out unless the server was started with --allow-writes.",
		Annotations:  &gomcp.ToolAnnotations{OpenWorldHint: new(false), DestructiveHint: new(true)},
		OutputSchema: mustOutputSchema[Built](),
	}, h.presetBuild)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "devices_list",
		Description:  "The Line 6 Helix hardware attached over USB. HX Edit must be quit for any tool that reaches the pedal.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[sdk.Attached](),
	}, h.devicesList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_list",
		Description:  "Every slot on the attached pedal and what it holds.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[sdk.Listing](),
	}, h.presetsList)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "preset_show",
		Description:  "One slot on the pedal, read back as a rig.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: true},
		OutputSchema: mustOutputSchema[Shown](),
	}, h.presetShow)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "preset_export",
		Description:  "Write one slot to a file: a rig by default, or the device's own .hlx with as=hlx. Refuses a file already at out unless the server was started with --allow-writes.",
		Annotations:  &gomcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: new(true)},
		OutputSchema: mustOutputSchema[sdk.Written](),
	}, h.presetExport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "preset_select",
		Description:  "Load a slot on the pedal, as pressing its footswitch does. Changes nothing stored.",
		Annotations:  &gomcp.ToolAnnotations{DestructiveHint: new(false), IdempotentHint: true},
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.presetSelect)

	if !allowWrites {
		return h.pedal
	}

	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "preset_import",
		Description:  "Put a .hlx into a slot on the pedal. Whatever the slot held is saved to a file first and then gone from the pedal.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.presetImport)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_copy",
		Description:  "Copy one slot onto another. The destination's old preset is saved to a file first.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.presetsCopy)
	gomcp.AddTool(s, &gomcp.Tool{
		Name:         "presets_swap",
		Description:  "Exchange two slots. Both are saved to files first. A slot holding no preset is refused, and nothing is written; use presets_copy to fill it.",
		Annotations:  destructive(),
		OutputSchema: mustOutputSchema[sdk.Change](),
	}, h.presetsSwap)

	return h.pedal
}

// mustOutputSchema infers T's output schema, correcting the types whose JSON
// reflection cannot see. It panics when T has no schema at all, which is a
// programming error found the moment the server starts.
//
// The SDK validates every structured result against this schema, so a schema
// narrower than what marshalling produces fails a real call rather than a test.
func mustOutputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](&jsonschema.ForOptions{TypeSchemas: outputTypeSchemas()})
	if err != nil {
		panic(fmt.Sprintf("mustOutputSchema[%s]: %v", reflect.TypeFor[T](), err))
	}

	return s
}

// outputTypeSchemas are the schemas reflection gets wrong on a tool's output.
func outputTypeSchemas() map[reflect.Type]*jsonschema.Schema {
	// anyJSON is every JSON value, spelled out rather than left as {}: the
	// inference adds "null" to a pointer's types, and added to an empty list
	// that would leave null as the only value allowed.
	anyJSON := &jsonschema.Schema{
		Types: []string{"null", "boolean", "number", "string", "array", "object"},
	}

	return map[reflect.Type]*jsonschema.Schema{
		// A ParamValue marshals to a bare number, string or bool depending on
		// a kind it keeps in unexported fields, so reflection sees an empty
		// struct and would demand an object.
		reflect.TypeFor[catalog.ParamValue](): {},
		// A RawMessage is a []byte to reflection, an array of small integers,
		// but it marshals as the JSON it holds. rig.Spec carries these for
		// device state it keeps without modelling.
		reflect.TypeFor[json.RawMessage](): anyJSON,
		// chain.Block.Attrs and rig.Spec's kept device fields. Nil on a
		// built chain, so it marshals to null, which a map's inferred
		// object-only schema refuses.
		reflect.TypeFor[map[string]json.RawMessage](): {
			Types:                []string{"null", "object"},
			AdditionalProperties: anyJSON,
		},
	}
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
