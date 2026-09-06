# Helix Preset Generator Phase 1 Implementation Plan

> **Superseded, 2026-09-06.** This plan targets a three-module layout that no
> longer exists, a `helixerr` package that has been removed, and golden-file
> byte comparison that cannot work. Tasks 1–8 were completed and their code
> lives on in `pkg/catalog` and `pkg/rig`; tasks 9 and 10 were never started and
> should be re-planned against the current layout. Kept as a record of what was
> built and why, not as instructions. See `CONTRIBUTING.md`.

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the deterministic core of `helix-core` — typed errors, the
parameter value union, the catalog and rig data models, and four validation
layers — plus a `helixctl` command that validates a rig against a catalog end to
end.

**Architecture:** Every input path converges on a `rig.Spec` that is validated
against a `catalog.Catalog` before anything writes a file. This plan builds that
spine bottom-up: errors, then values, then the two data models, then the
validators that relate them, then a CLI shell that holds no logic. The preset
writer and catalog extractor are deliberately excluded — both need real `.hlx`
exports that do not exist yet.

**Tech Stack:** Go 1.26, testify (suite + assert), cobra, `encoding/json`.

**Spec:** `docs/superpowers/specs/2026-09-03-helix-preset-generator-design.md`

## Global Constraints

- **Go 1.26** declared in `go.mod`. Do not lower it. `GOTOOLCHAIN=auto` fetches
  the toolchain on machines running older Go.
- **Coverage is gated at 100%.** `just test` fails below it. Every branch,
  including every error return, needs a test. Write the error-path test in the
  same step as the happy path.
- **Invoke tools through mise:** `mise exec -- just test`, never a bare `just`.
  A bare `just` resolves to whatever is installed globally.
- **Test file conventions:** `*_public_test.go` in the `{pkg}_test` package
  exercising the exported surface — this is the default. `*_test.go` in the same
  package only for what the exported surface cannot reach. Suite naming:
  `*_public_test.go` → `{Name}PublicTestSuite`, `*_test.go` → `{Name}TestSuite`.
- **Every exported identifier carries a doc comment starting with its own
  name.** Exactly one file per package carries the package comment.
- **File naming:** `snake_case.go`, one responsibility per file, named for it.
- **Take interfaces, return structs.** A validator that needs block lookup takes
  a `BlockLookup`, not a `*catalog.Catalog`.
- **Return errors, do not log them.** No `fmt.Print*`, no `log`, no `os.Exit`
  anywhere in `helix-core`. Wrap with `fmt.Errorf("...: %w", err)` only when
  adding context the caller cannot infer.
- **NO COMMITS BY AGENTS.** The repository owner commits. Each task ends by
  staging with `git add` and reporting what was staged. A suggested Conventional
  Commits subject is given for the owner's use.
- **Run `mise exec -- just ready` before reporting a task complete.** It adds
  licence headers, formats, vets and lints. A task is not done until it passes.
- **`.golangci.yml` sets `revive: enable-all-rules: true`.** That is far
  stricter than a default Go lint and will flag things the code in this plan
  does not anticipate — magic numbers, function length, cognitive complexity,
  argument counts. Expect to adjust. Fix the code where the rule has a point;
  where it does not, add a scoped `//nolint:revive // <reason>` naming the
  reason rather than loosening the config for the whole repository. Do not
  disable rules in `.golangci.yml` — it is a **managed** file and a
  `retemplate-go` run would silently revert the change.

______________________________________________________________________

## File Structure

Created in `helix-core/`:

| File                            | Responsibility                                                |
| ------------------------------- | ------------------------------------------------------------- |
| `pkg/helixerr/errors.go`        | Sentinel errors and the typed errors that wrap them           |
| `pkg/catalog/catalog.go`        | Package comment; `Catalog`, `Block`, `Param`, `DSPCost` types |
| `pkg/catalog/param_value.go`    | `ParamValue` tagged union and its JSON codec                  |
| `pkg/catalog/load.go`           | Reading a catalog from JSON, block lookup                     |
| `pkg/rig/rig.go`                | Package comment; `Spec`, `SpecBlock`, `Snapshot`, `Origin`    |
| `pkg/rig/limits.go`             | `Limits` — per-device ceilings the validators enforce         |
| `pkg/rig/validate_structure.go` | Layer 1 — every model exists                                  |
| `pkg/rig/validate_params.go`    | Layer 2 — keys exist, values fit type and range               |
| `pkg/rig/validate_budget.go`    | Layer 3 — DSP cost per chip                                   |
| `pkg/rig/validate_topology.go`  | Layer 4 — block count, positions, chip indices                |
| `pkg/source/source.go`          | The `Source` interface, `Request`, and the registry           |

Created in `helixctl/`:

| File                       | Responsibility            |
| -------------------------- | ------------------------- |
| `main.go`                  | Cobra root, wiring only   |
| `internal/cmd/validate.go` | The `validate` subcommand |

Deleted: `helix-core/pkg/helixcore/helixcore.go` — a generated stub whose
package name does not match the layout the spec describes. Go sources are
**seeded**, so removing it is safe and no retemplate will restore it.

**Not in this plan, and why.** Layer 5 (envelope validation), the synth writer,
the catalog extractor and the text source all require real `.hlx` exports to
establish the `device`, `version` and `modeldata_version` integers, whether
parameter values are normalised, and how snapshots are encoded. Building them
against guesses would produce code that compiles, passes its own tests, and
emits files the hardware rejects. They get their own plan once exports exist.

______________________________________________________________________

### Task 1: Typed errors and project dependencies

**Files:**

- Create: `helix-core/pkg/helixerr/errors.go`
- Create: `helix-core/pkg/helixerr/errors_public_test.go`
- Delete: `helix-core/pkg/helixcore/helixcore.go`
- Modify: `helix-core/go.mod` (adds testify)

**Interfaces:**

- Consumes: nothing — this is the base of the dependency graph.

- Produces: `helixerr.ErrUnknownBlock`, `ErrBadParam`, `ErrOverBudget`,
  `ErrBadTopology`, `ErrVersionMismatch`, `ErrNoMatch` (all `error`);
  `helixerr.UnknownBlockError{Model string}`,
  `helixerr.BadParamError{Model, Key, Reason string}`,
  `helixerr.OverBudgetError{Chip int, Cost, Ceiling float64}`,
  `helixerr.TopologyError{Reason string}` — each a struct with a pointer
  receiver `Error() string` and `Unwrap() error` returning its sentinel.

- [ ] **Step 1: Add testify and remove the generated stub**

```bash
cd helix-core
rm pkg/helixcore/helixcore.go
rmdir pkg/helixcore
mise exec -- go get github.com/stretchr/testify@latest
mise exec -- go mod tidy
```

- [ ] **Step 2: Write the failing test**

Create `pkg/helixerr/errors_public_test.go`:

```go
package helixerr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/helixerr"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

func (s *ErrorsPublicTestSuite) TestUnknownBlockErrorUnwrapsToSentinel() {
	err := &helixerr.UnknownBlockError{Model: "HD2_Nope"}

	s.Require().ErrorIs(err, helixerr.ErrUnknownBlock)
	s.Require().Contains(err.Error(), "HD2_Nope")
}

func (s *ErrorsPublicTestSuite) TestUnknownBlockErrorSurvivesWrapping() {
	err := fmt.Errorf("resolving rig: %w", &helixerr.UnknownBlockError{Model: "HD2_Nope"})

	s.Require().ErrorIs(err, helixerr.ErrUnknownBlock)

	var target *helixerr.UnknownBlockError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("HD2_Nope", target.Model)
}

func (s *ErrorsPublicTestSuite) TestBadParamErrorUnwrapsToSentinel() {
	err := &helixerr.BadParamError{Model: "HD2_AmpX", Key: "Gain", Reason: "out of range"}

	s.Require().ErrorIs(err, helixerr.ErrBadParam)
	s.Require().Contains(err.Error(), "Gain")
	s.Require().Contains(err.Error(), "out of range")
}

func (s *ErrorsPublicTestSuite) TestOverBudgetErrorUnwrapsToSentinel() {
	err := &helixerr.OverBudgetError{Chip: 1, Cost: 1.2, Ceiling: 0.95}

	s.Require().ErrorIs(err, helixerr.ErrOverBudget)
	s.Require().Contains(err.Error(), "chip 1")
}

func (s *ErrorsPublicTestSuite) TestTopologyErrorUnwrapsToSentinel() {
	err := &helixerr.TopologyError{Reason: "too many blocks"}

	s.Require().ErrorIs(err, helixerr.ErrBadTopology)
	s.Require().Contains(err.Error(), "too many blocks")
}

func (s *ErrorsPublicTestSuite) TestSentinelsAreDistinct() {
	all := []error{
		helixerr.ErrUnknownBlock,
		helixerr.ErrBadParam,
		helixerr.ErrOverBudget,
		helixerr.ErrBadTopology,
		helixerr.ErrVersionMismatch,
		helixerr.ErrNoMatch,
	}

	for i, a := range all {
		for j, b := range all {
			if i != j {
				s.Require().NotErrorIs(a, b)
			}
		}
	}
}

func TestErrorsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd helix-core && mise exec -- go test ./pkg/helixerr/... -v` Expected:
FAIL — package `helixerr` does not exist.

- [ ] **Step 4: Write the implementation**

Create `pkg/helixerr/errors.go`:

```go
// Package helixerr defines the error values that cross helix-core's package
// boundaries. Callers match on the sentinels with errors.Is and reach the
// detail with errors.As.
package helixerr

import (
	"errors"
	"fmt"
)

// ErrUnknownBlock reports a model identifier absent from the catalog.
var ErrUnknownBlock = errors.New("unknown block model")

// ErrBadParam reports a parameter that does not exist on a block, or whose
// value does not fit the declared type or range.
var ErrBadParam = errors.New("invalid parameter")

// ErrOverBudget reports a rig whose blocks exceed a DSP chip's ceiling.
var ErrOverBudget = errors.New("dsp budget exceeded")

// ErrBadTopology reports a rig whose block count, positions or chip
// assignments the device cannot represent.
var ErrBadTopology = errors.New("invalid topology")

// ErrVersionMismatch reports a catalog built for a different firmware than the
// one being targeted.
var ErrVersionMismatch = errors.New("catalog version mismatch")

// ErrNoMatch reports a source that could not resolve a request to a rig.
var ErrNoMatch = errors.New("no rig matched the request")

// UnknownBlockError names the model identifier that was not found.
type UnknownBlockError struct {
	Model string
}

// Error implements the error interface.
func (e *UnknownBlockError) Error() string {
	return fmt.Sprintf("unknown block model %q", e.Model)
}

// Unwrap returns ErrUnknownBlock so callers can match with errors.Is.
func (e *UnknownBlockError) Unwrap() error { return ErrUnknownBlock }

// BadParamError names the block, the parameter and why it was rejected.
type BadParamError struct {
	Model  string
	Key    string
	Reason string
}

// Error implements the error interface.
func (e *BadParamError) Error() string {
	return fmt.Sprintf("block %q parameter %q: %s", e.Model, e.Key, e.Reason)
}

// Unwrap returns ErrBadParam so callers can match with errors.Is.
func (e *BadParamError) Unwrap() error { return ErrBadParam }

// OverBudgetError names the chip that overflowed and by how much.
type OverBudgetError struct {
	Chip    int
	Cost    float64
	Ceiling float64
}

// Error implements the error interface.
func (e *OverBudgetError) Error() string {
	return fmt.Sprintf("chip %d costs %.3f, ceiling %.3f", e.Chip, e.Cost, e.Ceiling)
}

// Unwrap returns ErrOverBudget so callers can match with errors.Is.
func (e *OverBudgetError) Unwrap() error { return ErrOverBudget }

// TopologyError explains why a rig's shape is not representable.
type TopologyError struct {
	Reason string
}

// Error implements the error interface.
func (e *TopologyError) Error() string {
	return "invalid topology: " + e.Reason
}

// Unwrap returns ErrBadTopology so callers can match with errors.Is.
func (e *TopologyError) Unwrap() error { return ErrBadTopology }
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd helix-core && mise exec -- go test ./pkg/helixerr/... -v` Expected:
PASS, six tests.

- [ ] **Step 6: Run the full gate**

Run: `cd helix-core && mise exec -- just ready` Expected: format, lint and 100%
coverage all clean.

- [ ] **Step 7: Stage and report**

```bash
cd helix-core
git add pkg/helixerr go.mod go.sum
git status --short
```

Do not commit. Report what was staged. Suggested subject for the owner:
`feat(helixerr): add typed errors for core boundaries`

______________________________________________________________________

### Task 2: ParamValue tagged union

**Files:**

- Create: `helix-core/pkg/catalog/param_value.go`
- Create: `helix-core/pkg/catalog/param_value_public_test.go`

**Interfaces:**

- Consumes: `helixerr.ErrBadParam` from Task 1.
- Produces: `catalog.ParamType` (string) with constants `ParamFloat`,
  `ParamInt`, `ParamBool`, `ParamEnum`; `catalog.ParamValue` (struct, zero value
  invalid); constructors `Float(float64)`, `Int(int64)`, `Bool(bool)`,
  `Enum(string)` all returning `ParamValue`; accessors `Type() ParamType`,
  `Float() (float64, bool)`, `Int() (int64, bool)`, `Bool() (bool, bool)`,
  `Enum() (string, bool)`; `MarshalJSON() ([]byte, error)` on the value receiver
  and `UnmarshalJSON([]byte) error` on the pointer receiver.

Helix parameters mix floats, ints, bools and enum strings inside one JSON
object. `map[string]any` round-trips and discards type information at exactly
the boundary where it matters. A file that serialises a float where the device
expects an enum looks correct and does not load.

- [ ] **Step 1: Write the failing test**

Create `pkg/catalog/param_value_public_test.go`:

```go
package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
)

type ParamValuePublicTestSuite struct {
	suite.Suite
}

func (s *ParamValuePublicTestSuite) TestConstructorsSetType() {
	tests := []struct {
		name string
		got  catalog.ParamValue
		want catalog.ParamType
	}{
		{"float", catalog.Float(0.5), catalog.ParamFloat},
		{"int", catalog.Int(3), catalog.ParamInt},
		{"bool", catalog.Bool(true), catalog.ParamBool},
		{"enum", catalog.Enum("Normal"), catalog.ParamEnum},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Require().Equal(tc.want, tc.got.Type())
		})
	}
}

func (s *ParamValuePublicTestSuite) TestAccessorsReturnValueAndTrueOnMatch() {
	f, ok := catalog.Float(0.25).Float()
	s.Require().True(ok)
	s.Require().InDelta(0.25, f, 1e-9)

	i, ok := catalog.Int(7).Int()
	s.Require().True(ok)
	s.Require().Equal(int64(7), i)

	b, ok := catalog.Bool(true).Bool()
	s.Require().True(ok)
	s.Require().True(b)

	e, ok := catalog.Enum("Bright").Enum()
	s.Require().True(ok)
	s.Require().Equal("Bright", e)
}

func (s *ParamValuePublicTestSuite) TestAccessorsReturnFalseOnMismatch() {
	_, ok := catalog.Enum("Bright").Float()
	s.Require().False(ok)

	_, ok = catalog.Float(1).Int()
	s.Require().False(ok)

	_, ok = catalog.Int(1).Bool()
	s.Require().False(ok)

	_, ok = catalog.Bool(true).Enum()
	s.Require().False(ok)
}

func (s *ParamValuePublicTestSuite) TestMarshalRoundTripsEachType() {
	tests := []struct {
		name string
		val  catalog.ParamValue
		json string
	}{
		{"float", catalog.Float(0.5), "0.5"},
		{"int", catalog.Int(3), "3"},
		{"bool true", catalog.Bool(true), "true"},
		{"bool false", catalog.Bool(false), "false"},
		{"enum", catalog.Enum("Normal"), `"Normal"`},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			b, err := json.Marshal(tc.val)
			s.Require().NoError(err)
			s.Require().JSONEq(tc.json, string(b))

			var back catalog.ParamValue
			s.Require().NoError(json.Unmarshal(b, &back))
			s.Require().Equal(tc.val, back)
		})
	}
}

func (s *ParamValuePublicTestSuite) TestUnmarshalDistinguishesIntFromFloat() {
	var i catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5"), &i))
	s.Require().Equal(catalog.ParamInt, i.Type())

	var f catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5.0"), &f))
	s.Require().Equal(catalog.ParamFloat, f.Type())

	var e catalog.ParamValue
	s.Require().NoError(json.Unmarshal([]byte("5e2"), &e))
	s.Require().Equal(catalog.ParamFloat, e.Type())
}

func (s *ParamValuePublicTestSuite) TestMarshalZeroValueIsAnError() {
	var zero catalog.ParamValue

	_, err := json.Marshal(zero)
	s.Require().ErrorIs(err, helixerr.ErrBadParam)
}

func (s *ParamValuePublicTestSuite) TestUnmarshalRejectsMalformedInput() {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ``},
		{"object", `{"a":1}`},
		{"array", `[1,2]`},
		{"null", `null`},
		{"bad number", `1.2.3`},
		{"unterminated string", `"abc`},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var v catalog.ParamValue
			s.Require().Error(v.UnmarshalJSON([]byte(tc.in)))
		})
	}
}

func TestParamValuePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ParamValuePublicTestSuite))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd helix-core && mise exec -- go test ./pkg/catalog/... -v` Expected: FAIL
— package `catalog` does not exist.

- [ ] **Step 3: Write the implementation**

Create `pkg/catalog/param_value.go`:

```go
package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/retr0h/helix-core/pkg/helixerr"
)

// ParamType names the kind a ParamValue holds.
type ParamType string

// The parameter kinds a Helix block accepts.
const (
	ParamFloat ParamType = "float"
	ParamInt   ParamType = "int"
	ParamBool  ParamType = "bool"
	ParamEnum  ParamType = "enum"
)

// ParamValue is one parameter value of a known kind. Its zero value carries no
// kind and is invalid: marshalling it reports helixerr.ErrBadParam rather than
// silently emitting a value the device would reject.
type ParamValue struct {
	typ ParamType
	f   float64
	i   int64
	b   bool
	s   string
}

// Float returns a ParamValue holding v.
func Float(v float64) ParamValue { return ParamValue{typ: ParamFloat, f: v} }

// Int returns a ParamValue holding v.
func Int(v int64) ParamValue { return ParamValue{typ: ParamInt, i: v} }

// Bool returns a ParamValue holding v.
func Bool(v bool) ParamValue { return ParamValue{typ: ParamBool, b: v} }

// Enum returns a ParamValue holding the enumerated member v.
func Enum(v string) ParamValue { return ParamValue{typ: ParamEnum, s: v} }

// Type reports the kind held, or the empty ParamType for a zero value.
func (v ParamValue) Type() ParamType { return v.typ }

// Float returns the float held and whether the value holds one.
func (v ParamValue) Float() (float64, bool) { return v.f, v.typ == ParamFloat }

// Int returns the integer held and whether the value holds one.
func (v ParamValue) Int() (int64, bool) { return v.i, v.typ == ParamInt }

// Bool returns the boolean held and whether the value holds one.
func (v ParamValue) Bool() (bool, bool) { return v.b, v.typ == ParamBool }

// Enum returns the enumerated member held and whether the value holds one.
func (v ParamValue) Enum() (string, bool) { return v.s, v.typ == ParamEnum }

// MarshalJSON writes the value in its own kind. A zero ParamValue is an error.
func (v ParamValue) MarshalJSON() ([]byte, error) {
	switch v.typ {
	case ParamFloat:
		return json.Marshal(v.f)
	case ParamInt:
		return json.Marshal(v.i)
	case ParamBool:
		return json.Marshal(v.b)
	case ParamEnum:
		return json.Marshal(v.s)
	default:
		return nil, fmt.Errorf("%w: zero ParamValue has no kind", helixerr.ErrBadParam)
	}
}

// UnmarshalJSON infers the kind from the JSON literal. A number whose literal
// carries no '.', 'e' or 'E' is an integer; any other number is a float. JSON
// alone cannot distinguish 5 from 5.0 once decoded, so the literal decides.
func (v *ParamValue) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return fmt.Errorf("%w: empty value", helixerr.ErrBadParam)
	}

	switch {
	case s == "true", s == "false":
		v.typ, v.b = ParamBool, s == "true"

		return nil
	case s[0] == '"':
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return fmt.Errorf("%w: %w", helixerr.ErrBadParam, err)
		}

		v.typ, v.s = ParamEnum, str

		return nil
	case s[0] == '-' || (s[0] >= '0' && s[0] <= '9'):
		return v.unmarshalNumber(b, s)
	default:
		return fmt.Errorf("%w: %q is not a parameter value", helixerr.ErrBadParam, s)
	}
}

func (v *ParamValue) unmarshalNumber(b []byte, lit string) error {
	if !strings.ContainsAny(lit, ".eE") {
		var i int64
		if err := json.Unmarshal(b, &i); err != nil {
			return fmt.Errorf("%w: %w", helixerr.ErrBadParam, err)
		}

		v.typ, v.i = ParamInt, i

		return nil
	}

	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%w: %w", helixerr.ErrBadParam, err)
	}

	v.typ, v.f = ParamFloat, f

	return nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd helix-core && mise exec -- go test ./pkg/catalog/... -v` Expected:
PASS.

- [ ] **Step 5: Run the full gate**

Run: `cd helix-core && mise exec -- just ready` Expected: clean, coverage 100%
for the file.

- [ ] **Step 6: Stage and report**

```bash
cd helix-core
git add pkg/catalog
git status --short
```

Suggested subject: `feat(catalog): add ParamValue tagged union`

______________________________________________________________________

### Task 3: Catalog types and loader

**Files:**

- Create: `helix-core/pkg/catalog/catalog.go`
- Create: `helix-core/pkg/catalog/load.go`
- Create: `helix-core/pkg/catalog/load_public_test.go`
- Create: `helix-core/pkg/catalog/testdata/minimal.json`

**Interfaces:**

- Consumes: `catalog.ParamValue`, `catalog.ParamType` from Task 2.

- Produces: `catalog.ModelID` (string), `catalog.Category` (string) with
  constants `CategoryAmp`, `CategoryCab`, `CategoryDrive`, `CategoryComp`,
  `CategoryDelay`, `CategoryReverb`, `CategoryEQ`, `CategoryMod`,
  `CategoryOther`; `catalog.Provenance` (string) with `ProvObserved`,
  `ProvMeasured`, `ProvInherited`, `ProvAssumed`; structs `DSPCost`, `Param`,
  `Block`, `Catalog`; `catalog.Load(io.Reader) (*Catalog, error)`; method
  `(*Catalog).Block(ModelID) (Block, bool)`.

- [ ] **Step 1: Write the fixture**

Create `pkg/catalog/testdata/minimal.json`:

```json
{
  "device": "HX Stomp",
  "device_id": 2162689,
  "schema_version": 6,
  "modeldata_version": 30,
  "blocks": {
    "HD2_AmpTest": {
      "id": "HD2_AmpTest",
      "name": "Test Amp",
      "category": "amp",
      "stereo": false,
      "dsp": { "mono": 0.30, "stereo": 0.55, "prov": "measured" },
      "prov": "observed",
      "params": {
        "Gain": {
          "key": "Gain",
          "label": "Drive",
          "type": "float",
          "min": 0.0,
          "max": 1.0,
          "default": 0.5,
          "unit": ""
        },
        "Mode": {
          "key": "Mode",
          "label": "Mode",
          "type": "enum",
          "enum": ["Normal", "Bright"],
          "default": "Normal"
        }
      }
    }
  }
}
```

- [ ] **Step 2: Write the failing test**

Create `pkg/catalog/load_public_test.go`:

```go
package catalog_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
)

type LoadPublicTestSuite struct {
	suite.Suite
}

func (s *LoadPublicTestSuite) loadMinimal() *catalog.Catalog {
	s.T().Helper()

	f, err := os.Open("testdata/minimal.json")
	s.Require().NoError(err)

	defer func() { s.Require().NoError(f.Close()) }()

	c, err := catalog.Load(f)
	s.Require().NoError(err)

	return c
}

func (s *LoadPublicTestSuite) TestLoadReadsDeviceIdentity() {
	c := s.loadMinimal()

	s.Require().Equal("HX Stomp", c.Device)
	s.Require().Equal(2162689, c.DeviceID)
	s.Require().Equal(6, c.SchemaVersion)
	s.Require().Equal(30, c.ModelData)
}

func (s *LoadPublicTestSuite) TestLoadReadsBlockAndParams() {
	c := s.loadMinimal()

	b, ok := c.Block("HD2_AmpTest")
	s.Require().True(ok)
	s.Require().Equal("Test Amp", b.Name)
	s.Require().Equal(catalog.CategoryAmp, b.Category)
	s.Require().False(b.Stereo)
	s.Require().Equal(catalog.ProvObserved, b.Prov)
	s.Require().InDelta(0.30, b.DSP.Mono, 1e-9)
	s.Require().Equal(catalog.ProvMeasured, b.DSP.Prov)

	gain, ok := b.Params["Gain"]
	s.Require().True(ok)
	s.Require().Equal(catalog.ParamFloat, gain.Type)
	s.Require().InDelta(1.0, gain.Max, 1e-9)

	mode, ok := b.Params["Mode"]
	s.Require().True(ok)
	s.Require().Equal(catalog.ParamEnum, mode.Type)
	s.Require().Equal([]string{"Normal", "Bright"}, mode.Enum)
}

func (s *LoadPublicTestSuite) TestBlockReportsFalseForUnknownModel() {
	c := s.loadMinimal()

	_, ok := c.Block("HD2_Nope")
	s.Require().False(ok)
}

func (s *LoadPublicTestSuite) TestLoadRejectsMalformedJSON() {
	_, err := catalog.Load(strings.NewReader("{not json"))

	s.Require().Error(err)
}

func (s *LoadPublicTestSuite) TestLoadRejectsAReaderThatFails() {
	_, err := catalog.Load(&failingReader{})

	s.Require().Error(err)
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, os.ErrClosed }

func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run:
`cd helix-core && mise exec -- go test ./pkg/catalog/... -run LoadPublic -v`
Expected: FAIL — `catalog.Load` undefined.

- [ ] **Step 4: Write the types**

Create `pkg/catalog/catalog.go`:

```go
// Package catalog describes what a Helix device can do: which blocks exist,
// what parameters each accepts, and how much DSP each costs. It is derived by
// observing preset files exported from hardware, because the format has no
// published schema.
package catalog

// ModelID is a Line 6 internal model identifier, such as "HD2_AmpAmpegSVT".
type ModelID string

// Category groups blocks by what they do in a signal chain.
type Category string

// The block categories this catalog distinguishes.
const (
	CategoryAmp    Category = "amp"
	CategoryCab    Category = "cab"
	CategoryDrive  Category = "drive"
	CategoryComp   Category = "comp"
	CategoryDelay  Category = "delay"
	CategoryReverb Category = "reverb"
	CategoryEQ     Category = "eq"
	CategoryMod    Category = "mod"
	CategoryOther  Category = "other"
)

// Provenance records how a catalog entry came to be known. It gates delivery:
// nothing whose DSP cost is ProvAssumed may reach a user.
type Provenance string

// How a catalog entry was learned, most trustworthy first.
const (
	// ProvMeasured was swept on hardware. Ranges are trustworthy.
	ProvMeasured Provenance = "measured"
	// ProvObserved was seen in exported presets. Identifiers and parameter
	// keys are trustworthy; ranges are lower bounds, not true bounds.
	ProvObserved Provenance = "observed"
	// ProvInherited came from a third-party catalog and is unverified.
	ProvInherited Provenance = "inherited"
	// ProvAssumed is a guess. It never ships.
	ProvAssumed Provenance = "assumed"
)

// DSPCost is a block's share of one processor, expressed as a fraction of the
// chip. Stereo instances cost more than mono ones.
type DSPCost struct {
	Mono   float64    `json:"mono"`
	Stereo float64    `json:"stereo"`
	Prov   Provenance `json:"prov"`
}

// Param describes one parameter a block accepts.
type Param struct {
	Key     string     `json:"key"`
	Label   string     `json:"label"`
	Type    ParamType  `json:"type"`
	Min     float64    `json:"min"`
	Max     float64    `json:"max"`
	Default ParamValue `json:"default"`
	Enum    []string   `json:"enum,omitempty"`
	Unit    string     `json:"unit"`
}

// Block is one model the device can place in a signal chain.
type Block struct {
	ID       ModelID          `json:"id"`
	Name     string           `json:"name"`
	Category Category         `json:"category"`
	Params   map[string]Param `json:"params"`
	Stereo   bool             `json:"stereo"`
	DSP      DSPCost          `json:"dsp"`
	Prov     Provenance       `json:"prov"`
}

// Catalog is every block known for one device at one firmware version.
type Catalog struct {
	Device        string            `json:"device"`
	DeviceID      int               `json:"device_id"`
	SchemaVersion int               `json:"schema_version"`
	ModelData     int               `json:"modeldata_version"`
	Blocks        map[ModelID]Block `json:"blocks"`
}

// Block returns the block for id and whether the catalog holds one.
func (c *Catalog) Block(id ModelID) (Block, bool) {
	b, ok := c.Blocks[id]

	return b, ok
}
```

- [ ] **Step 5: Write the loader**

Create `pkg/catalog/load.go`:

```go
package catalog

import (
	"encoding/json"
	"fmt"
	"io"
)

// Load reads a catalog from JSON. It does not validate that the catalog is
// complete or internally consistent — it reports only what it could not parse.
func Load(r io.Reader) (*Catalog, error) {
	var c Catalog

	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, fmt.Errorf("decoding catalog: %w", err)
	}

	return &c, nil
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd helix-core && mise exec -- go test ./pkg/catalog/... -v` Expected:
PASS.

- [ ] **Step 7: Run the full gate**

Run: `cd helix-core && mise exec -- just ready`

- [ ] **Step 8: Stage and report**

```bash
cd helix-core
git add pkg/catalog
git status --short
```

Suggested subject: `feat(catalog): add catalog types and JSON loader`

______________________________________________________________________

### Task 4: Rig types and device limits

**Files:**

- Create: `helix-core/pkg/rig/rig.go`
- Create: `helix-core/pkg/rig/limits.go`
- Create: `helix-core/pkg/rig/rig_public_test.go`

**Interfaces:**

- Consumes: `catalog.ModelID`, `catalog.ParamValue` from Tasks 2 and 3.
- Produces: `rig.Origin` (string) with `OriginCurated`, `OriginLLM`,
  `OriginAudio`; structs `rig.SpecBlock`, `rig.Snapshot`, `rig.Spec`;
  `rig.Limits` struct with fields `MaxBlocks int`, `Chips int`,
  `ChipCeiling float64`; `rig.HXStompLimits() Limits`; interface
  `rig.BlockLookup` with method `Block(catalog.ModelID) (catalog.Block, bool)`.

`Origin` is **not** `catalog.Provenance`. Provenance records how a catalog entry
was learned; Origin records how a rig was decided. A rig has one Origin and
references blocks that each carry their own Provenance. Merging them loses the
ability to say "this DSP figure is a guess" and "this rig came from the model"
independently.

- [ ] **Step 1: Write the failing test**

Create `pkg/rig/rig_public_test.go`:

```go
package rig_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/rig"
)

type RigPublicTestSuite struct {
	suite.Suite
}

func (s *RigPublicTestSuite) TestSpecRoundTripsThroughJSON() {
	in := rig.Spec{
		Name:   "Test Rig",
		Origin: rig.OriginCurated,
		Blocks: []rig.SpecBlock{
			{
				Model:   "HD2_AmpTest",
				Params:  map[string]catalog.ParamValue{"Gain": catalog.Float(0.5)},
				DSP:     0,
				Pos:     0,
				Enabled: true,
			},
		},
		Snapshots: []rig.Snapshot{
			{
				Name:      "Lead",
				Overrides: map[int]map[string]catalog.ParamValue{0: {"Gain": catalog.Float(0.9)}},
			},
		},
	}

	b, err := json.Marshal(in)
	s.Require().NoError(err)

	var out rig.Spec
	s.Require().NoError(json.Unmarshal(b, &out))
	s.Require().Equal(in, out)
}

func (s *RigPublicTestSuite) TestHXStompLimitsAreTheDocumentedCeilings() {
	l := rig.HXStompLimits()

	s.Require().Equal(6, l.MaxBlocks)
	s.Require().Equal(2, l.Chips)
	s.Require().InDelta(0.95, l.ChipCeiling, 1e-9)
}

func (s *RigPublicTestSuite) TestOriginsAreDistinct() {
	s.Require().NotEqual(rig.OriginCurated, rig.OriginLLM)
	s.Require().NotEqual(rig.OriginLLM, rig.OriginAudio)
	s.Require().NotEqual(rig.OriginCurated, rig.OriginAudio)
}

func TestRigPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RigPublicTestSuite))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd helix-core && mise exec -- go test ./pkg/rig/... -v` Expected: FAIL —
package `rig` does not exist.

- [ ] **Step 3: Write the types**

Create `pkg/rig/rig.go`:

```go
// Package rig describes a signal chain as intended, independent of the file
// format that will carry it. Every input path produces a Spec; everything that
// writes files consumes one.
package rig

import "github.com/retr0h/helix-core/pkg/catalog"

// Origin records how a rig was decided. It is distinct from
// catalog.Provenance, which records how a catalog entry was learned.
type Origin string

// How a rig was decided.
const (
	// OriginCurated matched a hand-authored recipe.
	OriginCurated Origin = "curated"
	// OriginLLM was generated against the catalog schema.
	OriginLLM Origin = "llm"
	// OriginAudio was derived from measurement.
	OriginAudio Origin = "audio"
)

// SpecBlock is one block placed in a chain: which model, at which position on
// which processor, with which parameters.
type SpecBlock struct {
	Model   catalog.ModelID               `json:"model"`
	Params  map[string]catalog.ParamValue `json:"params"`
	DSP     int                           `json:"dsp"`
	Pos     int                           `json:"pos"`
	Enabled bool                          `json:"enabled"`
}

// Snapshot is a named set of parameter overrides, keyed by the index of the
// block in Spec.Blocks that each override applies to.
type Snapshot struct {
	Name      string                                   `json:"name"`
	Overrides map[int]map[string]catalog.ParamValue    `json:"overrides"`
}

// Spec is a complete signal chain, validated against a catalog before use.
type Spec struct {
	Name      string      `json:"name"`
	Blocks    []SpecBlock `json:"blocks"`
	Snapshots []Snapshot  `json:"snapshots,omitempty"`
	Origin    Origin      `json:"origin"`
}

// BlockLookup resolves a model identifier to its catalog entry. *catalog.Catalog
// satisfies it. Validators take this rather than a concrete catalog so a test
// can supply a handful of blocks without building a file.
type BlockLookup interface {
	Block(catalog.ModelID) (catalog.Block, bool)
}
```

- [ ] **Step 4: Write the limits**

Create `pkg/rig/limits.go`:

```go
package rig

// Limits are the ceilings one device imposes on a rig.
type Limits struct {
	// MaxBlocks is the most blocks the device will hold.
	MaxBlocks int
	// Chips is how many DSP processors the device has.
	Chips int
	// ChipCeiling is the fraction of one processor a rig may occupy. It is
	// deliberately below 1.0: a rig that exactly fills a chip in theory is a
	// rig that fails to load in practice.
	ChipCeiling float64
}

// HXStompLimits returns the ceilings for a Line 6 HX Stomp.
//
// These figures are UNVERIFIED against hardware. The block count and chip
// count come from published specifications; the ceiling is a chosen safety
// margin, not a measurement. Confirm all three against a real device before
// any generated preset reaches a user.
func HXStompLimits() Limits {
	return Limits{
		MaxBlocks:   6,
		Chips:       2,
		ChipCeiling: 0.95,
	}
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd helix-core && mise exec -- go test ./pkg/rig/... -v` Expected: PASS.

- [ ] **Step 6: Run the full gate, then stage**

```bash
cd helix-core
mise exec -- just ready
git add pkg/rig
git status --short
```

Suggested subject: `feat(rig): add Spec types and device limits`

______________________________________________________________________

### Task 5: Structural validation

**Files:**

- Create: `helix-core/pkg/rig/validate_structure.go`
- Create: `helix-core/pkg/rig/validate_structure_public_test.go`
- Create: `helix-core/pkg/rig/fake_lookup_test.go`

**Interfaces:**

- Consumes: `rig.Spec`, `rig.BlockLookup` from Task 4;
  `helixerr.UnknownBlockError` from Task 1.

- Produces: `rig.ValidateStructure(BlockLookup, Spec) error`.

- [ ] **Step 1: Write the test fake**

A fake, not a mock — the contributing guide prefers a real implementation over a
fake and a fake over a mock, and a mock asserting call order would test the
implementation rather than the behaviour.

Create `pkg/rig/fake_lookup_test.go`:

```go
package rig_test

import "github.com/retr0h/helix-core/pkg/catalog"

// fakeLookup is an in-memory BlockLookup for tests.
type fakeLookup struct {
	blocks map[catalog.ModelID]catalog.Block
}

func (f *fakeLookup) Block(id catalog.ModelID) (catalog.Block, bool) {
	b, ok := f.blocks[id]

	return b, ok
}

// testAmp returns a block with one float and one enum parameter.
func testAmp() catalog.Block {
	return catalog.Block{
		ID:       "HD2_AmpTest",
		Name:     "Test Amp",
		Category: catalog.CategoryAmp,
		Stereo:   false,
		Prov:     catalog.ProvObserved,
		DSP:      catalog.DSPCost{Mono: 0.30, Stereo: 0.55, Prov: catalog.ProvMeasured},
		Params: map[string]catalog.Param{
			"Gain": {
				Key: "Gain", Label: "Drive", Type: catalog.ParamFloat,
				Min: 0.0, Max: 1.0, Default: catalog.Float(0.5),
			},
			"Mode": {
				Key: "Mode", Label: "Mode", Type: catalog.ParamEnum,
				Enum: []string{"Normal", "Bright"}, Default: catalog.Enum("Normal"),
			},
		},
	}
}

func newFake(blocks ...catalog.Block) *fakeLookup {
	m := make(map[catalog.ModelID]catalog.Block, len(blocks))
	for _, b := range blocks {
		m[b.ID] = b
	}

	return &fakeLookup{blocks: m}
}
```

- [ ] **Step 2: Write the failing test**

Create `pkg/rig/validate_structure_public_test.go`:

```go
package rig_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helix-core/pkg/rig"
)

type ValidateStructurePublicTestSuite struct {
	suite.Suite
}

func (s *ValidateStructurePublicTestSuite) TestAcceptsARigOfKnownBlocks() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest"}}}

	s.Require().NoError(rig.ValidateStructure(newFake(testAmp()), spec))
}

func (s *ValidateStructurePublicTestSuite) TestAcceptsAnEmptyRig() {
	s.Require().NoError(rig.ValidateStructure(newFake(), rig.Spec{}))
}

func (s *ValidateStructurePublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest"},
		{Model: "HD2_Nope"},
	}}

	err := rig.ValidateStructure(newFake(testAmp()), spec)

	s.Require().ErrorIs(err, helixerr.ErrUnknownBlock)

	var target *helixerr.UnknownBlockError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("HD2_Nope", target.Model)
}

func TestValidateStructurePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateStructurePublicTestSuite))
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run:
`cd helix-core && mise exec -- go test ./pkg/rig/... -run ValidateStructure -v`
Expected: FAIL — `rig.ValidateStructure` undefined.

- [ ] **Step 4: Write the implementation**

Create `pkg/rig/validate_structure.go`:

```go
package rig

import "github.com/retr0h/helix-core/pkg/helixerr"

// ValidateStructure reports the first block in s whose model the catalog does
// not hold. It is the first of the validation layers and answers only one
// question, so a failure names one cause.
func ValidateStructure(l BlockLookup, s Spec) error {
	for _, b := range s.Blocks {
		if _, ok := l.Block(b.Model); !ok {
			return &helixerr.UnknownBlockError{Model: string(b.Model)}
		}
	}

	return nil
}
```

- [ ] **Step 5: Run the test, run the gate, stage**

```bash
cd helix-core
mise exec -- go test ./pkg/rig/... -v
mise exec -- just ready
git add pkg/rig
git status --short
```

Suggested subject: `feat(rig): add structural validation`

______________________________________________________________________

### Task 6: Parametric validation

**Files:**

- Create: `helix-core/pkg/rig/validate_params.go`
- Create: `helix-core/pkg/rig/validate_params_public_test.go`

**Interfaces:**

- Consumes: `rig.BlockLookup`, `rig.Spec`, `catalog.ParamValue`,
  `helixerr.BadParamError`, `helixerr.UnknownBlockError`.
- Produces: `rig.ValidateParams(BlockLookup, Spec) error`.

Parameters are iterated in sorted key order. Go's map iteration order is
randomised, and a validator that reports a different one of several bad
parameters on each run is untestable and miserable to debug.

- [ ] **Step 1: Write the failing test**

Create `pkg/rig/validate_params_public_test.go`:

```go
package rig_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helix-core/pkg/rig"
)

type ValidateParamsPublicTestSuite struct {
	suite.Suite
}

func (s *ValidateParamsPublicTestSuite) specWith(params map[string]catalog.ParamValue) rig.Spec {
	return rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", Params: params}}}
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsValuesInRange() {
	spec := s.specWith(map[string]catalog.ParamValue{
		"Gain": catalog.Float(0.5),
		"Mode": catalog.Enum("Bright"),
	})

	s.Require().NoError(rig.ValidateParams(newFake(testAmp()), spec))
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsValuesOnTheBoundary() {
	for _, v := range []float64{0.0, 1.0} {
		spec := s.specWith(map[string]catalog.ParamValue{"Gain": catalog.Float(v)})
		s.Require().NoError(rig.ValidateParams(newFake(testAmp()), spec))
	}
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnUnknownParameter() {
	spec := s.specWith(map[string]catalog.ParamValue{"Nope": catalog.Float(0.5)})

	err := rig.ValidateParams(newFake(testAmp()), spec)

	s.Require().ErrorIs(err, helixerr.ErrBadParam)

	var target *helixerr.BadParamError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal("Nope", target.Key)
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAWrongType() {
	spec := s.specWith(map[string]catalog.ParamValue{"Gain": catalog.Enum("loud")})

	err := rig.ValidateParams(newFake(testAmp()), spec)

	s.Require().ErrorIs(err, helixerr.ErrBadParam)
	s.Require().Contains(err.Error(), "expected float")
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAFloatOutOfRange() {
	for _, v := range []float64{-0.01, 1.01} {
		spec := s.specWith(map[string]catalog.ParamValue{"Gain": catalog.Float(v)})

		err := rig.ValidateParams(newFake(testAmp()), spec)

		s.Require().ErrorIs(err, helixerr.ErrBadParam)
		s.Require().Contains(err.Error(), "out of range")
	}
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnIntOutOfRange() {
	blk := testAmp()
	blk.Params["Taps"] = catalog.Param{
		Key: "Taps", Type: catalog.ParamInt, Min: 1, Max: 4, Default: catalog.Int(1),
	}

	spec := s.specWith(map[string]catalog.ParamValue{"Taps": catalog.Int(9)})

	err := rig.ValidateParams(newFake(blk), spec)

	s.Require().ErrorIs(err, helixerr.ErrBadParam)
}

func (s *ValidateParamsPublicTestSuite) TestAcceptsABoolWithoutRangeChecking() {
	blk := testAmp()
	blk.Params["Bright"] = catalog.Param{
		Key: "Bright", Type: catalog.ParamBool, Default: catalog.Bool(false),
	}

	spec := s.specWith(map[string]catalog.ParamValue{"Bright": catalog.Bool(true)})

	s.Require().NoError(rig.ValidateParams(newFake(blk), spec))
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnEnumMemberNotDeclared() {
	spec := s.specWith(map[string]catalog.ParamValue{"Mode": catalog.Enum("Sparkle")})

	err := rig.ValidateParams(newFake(testAmp()), spec)

	s.Require().ErrorIs(err, helixerr.ErrBadParam)
	s.Require().Contains(err.Error(), "Sparkle")
}

func (s *ValidateParamsPublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_Nope"}}}

	err := rig.ValidateParams(newFake(testAmp()), spec)

	s.Require().ErrorIs(err, helixerr.ErrUnknownBlock)
}

func (s *ValidateParamsPublicTestSuite) TestReportsTheFirstBadParameterInSortedOrder() {
	spec := s.specWith(map[string]catalog.ParamValue{
		"Zebra": catalog.Float(0.5),
		"Alpha": catalog.Float(0.5),
	})

	for range 20 {
		err := rig.ValidateParams(newFake(testAmp()), spec)

		var target *helixerr.BadParamError
		s.Require().True(errors.As(err, &target))
		s.Require().Equal("Alpha", target.Key, "must be deterministic across runs")
	}
}

func TestValidateParamsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateParamsPublicTestSuite))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
`cd helix-core && mise exec -- go test ./pkg/rig/... -run ValidateParams -v`
Expected: FAIL — `rig.ValidateParams` undefined.

- [ ] **Step 3: Write the implementation**

Create `pkg/rig/validate_params.go`:

```go
package rig

import (
	"fmt"
	"maps"
	"slices"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
)

// ValidateParams reports the first parameter in s that the catalog does not
// declare, or whose value does not fit the declared kind or range. Parameters
// are checked in sorted key order so the same rig always reports the same
// failure.
func ValidateParams(l BlockLookup, s Spec) error {
	for _, sb := range s.Blocks {
		blk, ok := l.Block(sb.Model)
		if !ok {
			return &helixerr.UnknownBlockError{Model: string(sb.Model)}
		}

		for _, key := range slices.Sorted(maps.Keys(sb.Params)) {
			if err := checkParam(blk, key, sb.Params[key]); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkParam(blk catalog.Block, key string, val catalog.ParamValue) error {
	p, ok := blk.Params[key]
	if !ok {
		return &helixerr.BadParamError{
			Model: string(blk.ID), Key: key, Reason: "no such parameter",
		}
	}

	if val.Type() != p.Type {
		return &helixerr.BadParamError{
			Model: string(blk.ID), Key: key,
			Reason: fmt.Sprintf("expected %s, got %s", p.Type, val.Type()),
		}
	}

	switch p.Type {
	case catalog.ParamFloat:
		f, _ := val.Float()

		return checkRange(blk, key, f, p)
	case catalog.ParamInt:
		i, _ := val.Int()

		return checkRange(blk, key, float64(i), p)
	case catalog.ParamEnum:
		e, _ := val.Enum()
		if !slices.Contains(p.Enum, e) {
			return &helixerr.BadParamError{
				Model: string(blk.ID), Key: key,
				Reason: fmt.Sprintf("%q is not a declared member", e),
			}
		}
	case catalog.ParamBool:
		// A bool has no range to check.
	}

	return nil
}

func checkRange(blk catalog.Block, key string, v float64, p catalog.Param) error {
	if v < p.Min || v > p.Max {
		return &helixerr.BadParamError{
			Model: string(blk.ID), Key: key,
			Reason: fmt.Sprintf("%v out of range [%v, %v]", v, p.Min, p.Max),
		}
	}

	return nil
}
```

- [ ] **Step 4: Run the test, run the gate, stage**

```bash
cd helix-core
mise exec -- go test ./pkg/rig/... -v
mise exec -- just ready
git add pkg/rig
git status --short
```

Suggested subject: `feat(rig): add parametric validation`

______________________________________________________________________

### Task 7: DSP budget validation

**Files:**

- Create: `helix-core/pkg/rig/validate_budget.go`
- Create: `helix-core/pkg/rig/validate_budget_public_test.go`

**Interfaces:**

- Consumes: `rig.BlockLookup`, `rig.Spec`, `rig.Limits`, `catalog.ProvAssumed`,
  `helixerr.OverBudgetError`, `helixerr.TopologyError`,
  `helixerr.BadParamError`.
- Produces: `rig.ValidateBudget(BlockLookup, Spec, Limits) error`.

Two behaviours that are easy to get wrong and are pinned by tests:

**Bypassed blocks still cost DSP.** On Helix hardware a disabled block occupies
its processor. Skipping `Enabled == false` would let a rig pass validation and
fail to load.

**An assumed DSP cost is refused outright.** The spec makes this a hard gate:
DSP figures are the least reliable data in the system, and an over-budget preset
that will not load is the most visible way this product fails.

- [ ] **Step 1: Write the failing test**

Create `pkg/rig/validate_budget_public_test.go`:

```go
package rig_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helix-core/pkg/rig"
)

type ValidateBudgetPublicTestSuite struct {
	suite.Suite
}

func (s *ValidateBudgetPublicTestSuite) limits() rig.Limits {
	return rig.Limits{MaxBlocks: 6, Chips: 2, ChipCeiling: 0.95}
}

func (s *ValidateBudgetPublicTestSuite) TestAcceptsARigInsideTheCeiling() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 1, Enabled: true},
	}}

	s.Require().NoError(rig.ValidateBudget(newFake(testAmp()), spec, s.limits()))
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAChipOverTheCeiling() {
	// testAmp costs 0.30 mono; four on one chip is 1.20, over 0.95.
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	err := rig.ValidateBudget(newFake(testAmp()), spec, s.limits())

	s.Require().ErrorIs(err, helixerr.ErrOverBudget)

	var target *helixerr.OverBudgetError
	s.Require().True(errors.As(err, &target))
	s.Require().Equal(0, target.Chip)
	s.Require().InDelta(1.20, target.Cost, 1e-9)
}

func (s *ValidateBudgetPublicTestSuite) TestCountsBypassedBlocks() {
	// Same four blocks, all disabled. Hardware still spends the DSP.
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: false},
	}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(testAmp()), spec, s.limits()),
		helixerr.ErrOverBudget,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestChargesStereoBlocksTheStereoCost() {
	blk := testAmp()
	blk.Stereo = true // 0.55 each; two is 1.10, over 0.95

	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
		{Model: "HD2_AmpTest", DSP: 0, Enabled: true},
	}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(blk), spec, s.limits()),
		helixerr.ErrOverBudget,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRefusesAnAssumedDSPCost() {
	blk := testAmp()
	blk.DSP.Prov = catalog.ProvAssumed

	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: 0}}}

	err := rig.ValidateBudget(newFake(blk), spec, s.limits())

	s.Require().ErrorIs(err, helixerr.ErrBadParam)
	s.Require().Contains(err.Error(), "assumed")
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: 5}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(testAmp()), spec, s.limits()),
		helixerr.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsANegativeChipIndex() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest", DSP: -1}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(testAmp()), spec, s.limits()),
		helixerr.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsLimitsDeclaringNoChips() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_AmpTest"}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(testAmp()), spec, rig.Limits{Chips: 0}),
		helixerr.ErrBadTopology,
	)
}

func (s *ValidateBudgetPublicTestSuite) TestRejectsAnUnknownModel() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "HD2_Nope"}}}

	s.Require().ErrorIs(
		rig.ValidateBudget(newFake(testAmp()), spec, s.limits()),
		helixerr.ErrUnknownBlock,
	)
}

func TestValidateBudgetPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateBudgetPublicTestSuite))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
`cd helix-core && mise exec -- go test ./pkg/rig/... -run ValidateBudget -v`
Expected: FAIL — `rig.ValidateBudget` undefined.

- [ ] **Step 3: Write the implementation**

Create `pkg/rig/validate_budget.go`:

```go
package rig

import (
	"fmt"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
)

// ValidateBudget reports the first DSP processor whose blocks exceed the
// ceiling in lim.
//
// Bypassed blocks are counted. On Helix hardware a disabled block still
// occupies its processor, so skipping them would pass a rig that will not
// load.
//
// A block whose DSP cost is catalog.ProvAssumed is refused outright rather
// than counted. An assumed figure cannot support a claim that a rig fits.
func ValidateBudget(l BlockLookup, s Spec, lim Limits) error {
	if lim.Chips <= 0 {
		return &helixerr.TopologyError{Reason: "limits declare no dsp processors"}
	}

	costs := make([]float64, lim.Chips)

	for _, sb := range s.Blocks {
		blk, ok := l.Block(sb.Model)
		if !ok {
			return &helixerr.UnknownBlockError{Model: string(sb.Model)}
		}

		if sb.DSP < 0 || sb.DSP >= lim.Chips {
			return &helixerr.TopologyError{
				Reason: fmt.Sprintf(
					"block %q is on processor %d, device has %d",
					sb.Model, sb.DSP, lim.Chips,
				),
			}
		}

		if blk.DSP.Prov == catalog.ProvAssumed {
			return &helixerr.BadParamError{
				Model: string(sb.Model), Key: "dsp",
				Reason: "cost is assumed and must not reach a user",
			}
		}

		cost := blk.DSP.Mono
		if blk.Stereo {
			cost = blk.DSP.Stereo
		}

		costs[sb.DSP] += cost
	}

	for chip, c := range costs {
		if c > lim.ChipCeiling {
			return &helixerr.OverBudgetError{Chip: chip, Cost: c, Ceiling: lim.ChipCeiling}
		}
	}

	return nil
}
```

- [ ] **Step 4: Run the test, run the gate, stage**

```bash
cd helix-core
mise exec -- go test ./pkg/rig/... -v
mise exec -- just ready
git add pkg/rig
git status --short
```

Suggested subject: `feat(rig): add DSP budget validation`

______________________________________________________________________

### Task 8: Topological validation and the composed check

**Files:**

- Create: `helix-core/pkg/rig/validate_topology.go`
- Create: `helix-core/pkg/rig/validate.go`
- Create: `helix-core/pkg/rig/validate_topology_public_test.go`
- Create: `helix-core/pkg/rig/validate_public_test.go`

**Interfaces:**

- Consumes: everything from Tasks 4 through 7.

- Produces: `rig.ValidateTopology(Spec, Limits) error`;
  `rig.Validate(BlockLookup, Spec, Limits) error` running all four layers in
  order.

- [ ] **Step 1: Write the failing tests**

Create `pkg/rig/validate_topology_public_test.go`:

```go
package rig_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helix-core/pkg/rig"
)

type ValidateTopologyPublicTestSuite struct {
	suite.Suite
}

func (s *ValidateTopologyPublicTestSuite) limits() rig.Limits {
	return rig.Limits{MaxBlocks: 6, Chips: 2, ChipCeiling: 0.95}
}

func (s *ValidateTopologyPublicTestSuite) TestAcceptsContiguousPositionsPerChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 1},
		{Model: "C", DSP: 1, Pos: 0},
	}}

	s.Require().NoError(rig.ValidateTopology(spec, s.limits()))
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAnEmptyRig() {
	err := rig.ValidateTopology(rig.Spec{}, s.limits())

	s.Require().ErrorIs(err, helixerr.ErrBadTopology)
	s.Require().Contains(err.Error(), "no blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsTooManyBlocks() {
	blocks := make([]rig.SpecBlock, 7)
	for i := range blocks {
		blocks[i] = rig.SpecBlock{Model: "A", DSP: 0, Pos: i}
	}

	err := rig.ValidateTopology(rig.Spec{Blocks: blocks}, s.limits())

	s.Require().ErrorIs(err, helixerr.ErrBadTopology)
	s.Require().Contains(err.Error(), "7 blocks")
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsDuplicatePositionsOnOneChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 0},
	}}

	s.Require().ErrorIs(rig.ValidateTopology(spec, s.limits()), helixerr.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsAGapInPositions() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{
		{Model: "A", DSP: 0, Pos: 0},
		{Model: "B", DSP: 0, Pos: 2},
	}}

	s.Require().ErrorIs(rig.ValidateTopology(spec, s.limits()), helixerr.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsANegativePosition() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: -1}}}

	s.Require().ErrorIs(rig.ValidateTopology(spec, s.limits()), helixerr.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsABlockOnANonexistentChip() {
	spec := rig.Spec{Blocks: []rig.SpecBlock{{Model: "A", DSP: 9, Pos: 0}}}

	s.Require().ErrorIs(rig.ValidateTopology(spec, s.limits()), helixerr.ErrBadTopology)
}

func (s *ValidateTopologyPublicTestSuite) TestRejectsASnapshotOverridingAMissingBlock() {
	spec := rig.Spec{
		Blocks: []rig.SpecBlock{{Model: "A", DSP: 0, Pos: 0}},
		Snapshots: []rig.Snapshot{{
			Name:      "Lead",
			Overrides: map[int]map[string]catalog.ParamValue{4: {"Gain": catalog.Float(0.9)}},
		}},
	}

	err := rig.ValidateTopology(spec, s.limits())

	s.Require().ErrorIs(err, helixerr.ErrBadTopology)
	s.Require().Contains(err.Error(), "snapshot")
}

func TestValidateTopologyPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidateTopologyPublicTestSuite))
}
```

Create `pkg/rig/validate_public_test.go`:

```go
package rig_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helix-core/pkg/rig"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func (s *ValidatePublicTestSuite) limits() rig.Limits {
	return rig.Limits{MaxBlocks: 6, Chips: 2, ChipCeiling: 0.95}
}

func (s *ValidatePublicTestSuite) TestAcceptsAValidRig() {
	spec := rig.Spec{
		Name:   "Fine",
		Origin: rig.OriginCurated,
		Blocks: []rig.SpecBlock{{
			Model:   "HD2_AmpTest",
			Params:  map[string]catalog.ParamValue{"Gain": catalog.Float(0.5)},
			DSP:     0,
			Pos:     0,
			Enabled: true,
		}},
	}

	s.Require().NoError(rig.Validate(newFake(testAmp()), spec, s.limits()))
}

func (s *ValidatePublicTestSuite) TestReportsStructureBeforeParams() {
	// Unknown model AND a bad parameter. Structure runs first, so the caller
	// learns the model is unknown rather than that a parameter is missing from
	// a block that does not exist.
	spec := rig.Spec{Blocks: []rig.SpecBlock{{
		Model:  "HD2_Nope",
		Params: map[string]catalog.ParamValue{"Whatever": catalog.Float(99)},
		Pos:    0,
	}}}

	s.Require().ErrorIs(
		rig.Validate(newFake(testAmp()), spec, s.limits()),
		helixerr.ErrUnknownBlock,
	)
}

func (s *ValidatePublicTestSuite) TestReportsTopologyBeforeBudget() {
	// Seven blocks would also blow the budget. Topology runs first because
	// "too many blocks" is the more actionable message.
	blocks := make([]rig.SpecBlock, 7)
	for i := range blocks {
		blocks[i] = rig.SpecBlock{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	s.Require().ErrorIs(
		rig.Validate(newFake(testAmp()), rig.Spec{Blocks: blocks}, s.limits()),
		helixerr.ErrBadTopology,
	)
}

func (s *ValidatePublicTestSuite) TestReportsBudgetLast() {
	blocks := make([]rig.SpecBlock, 4)
	for i := range blocks {
		blocks[i] = rig.SpecBlock{Model: "HD2_AmpTest", DSP: 0, Pos: i}
	}

	s.Require().ErrorIs(
		rig.Validate(newFake(testAmp()), rig.Spec{Blocks: blocks}, s.limits()),
		helixerr.ErrOverBudget,
	)
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd helix-core && mise exec -- go test ./pkg/rig/... -v` Expected: FAIL —
`rig.ValidateTopology` and `rig.Validate` undefined.

- [ ] **Step 3: Write the topology implementation**

Create `pkg/rig/validate_topology.go`:

```go
package rig

import (
	"fmt"
	"maps"
	"slices"

	"github.com/retr0h/helix-core/pkg/helixerr"
)

// ValidateTopology reports a rig whose shape the device cannot represent:
// too many blocks, a processor that does not exist, or positions on one
// processor that are not the contiguous run 0..n-1.
//
// It needs no catalog — every question it answers is about the rig alone.
func ValidateTopology(s Spec, lim Limits) error {
	if len(s.Blocks) == 0 {
		return &helixerr.TopologyError{Reason: "rig has no blocks"}
	}

	if len(s.Blocks) > lim.MaxBlocks {
		return &helixerr.TopologyError{
			Reason: fmt.Sprintf("%d blocks, device holds %d", len(s.Blocks), lim.MaxBlocks),
		}
	}

	byChip := make(map[int][]int)

	for _, b := range s.Blocks {
		if b.DSP < 0 || b.DSP >= lim.Chips {
			return &helixerr.TopologyError{
				Reason: fmt.Sprintf(
					"block %q is on processor %d, device has %d",
					b.Model, b.DSP, lim.Chips,
				),
			}
		}

		if b.Pos < 0 {
			return &helixerr.TopologyError{
				Reason: fmt.Sprintf("block %q has negative position %d", b.Model, b.Pos),
			}
		}

		byChip[b.DSP] = append(byChip[b.DSP], b.Pos)
	}

	for _, chip := range slices.Sorted(maps.Keys(byChip)) {
		positions := byChip[chip]
		slices.Sort(positions)

		for i, p := range positions {
			if p != i {
				return &helixerr.TopologyError{
					Reason: fmt.Sprintf(
						"processor %d positions are not contiguous from zero: %v",
						chip, positions,
					),
				}
			}
		}
	}

	return validateSnapshots(s)
}

func validateSnapshots(s Spec) error {
	for _, snap := range s.Snapshots {
		for _, idx := range slices.Sorted(maps.Keys(snap.Overrides)) {
			if idx < 0 || idx >= len(s.Blocks) {
				return &helixerr.TopologyError{
					Reason: fmt.Sprintf(
						"snapshot %q overrides block %d, rig has %d",
						snap.Name, idx, len(s.Blocks),
					),
				}
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Write the composed check**

Create `pkg/rig/validate.go`:

```go
package rig

// Validate runs every layer in the order that produces the most useful first
// failure: structure, then parameters, then topology, then budget.
//
// The order is deliberate. A rig naming a model that does not exist should say
// so rather than complain that a parameter is missing from a block that was
// never found; a rig with too many blocks should say that rather than report
// the DSP overflow that follows from it.
//
// Callers wanting to know every problem at once should call the layers
// individually. This returns the first.
func Validate(l BlockLookup, s Spec, lim Limits) error {
	if err := ValidateStructure(l, s); err != nil {
		return err
	}

	if err := ValidateParams(l, s); err != nil {
		return err
	}

	if err := ValidateTopology(s, lim); err != nil {
		return err
	}

	return ValidateBudget(l, s, lim)
}
```

- [ ] **Step 5: Run the tests, run the gate, stage**

```bash
cd helix-core
mise exec -- go test ./pkg/rig/... -v
mise exec -- just ready
git add pkg/rig
git status --short
```

Suggested subject: `feat(rig): add topology validation and composed check`

______________________________________________________________________

### Task 9: The Source interface and registry

**Files:**

- Create: `helix-core/pkg/source/source.go`
- Create: `helix-core/pkg/source/registry.go`
- Create: `helix-core/pkg/source/registry_public_test.go`

**Interfaces:**

- Consumes: `rig.Spec`, `helixerr.ErrNoMatch`.
- Produces: `source.Request` struct with fields `Text string`, `Device string`;
  `source.Source` interface with `Name() string` and
  `Resolve(context.Context, Request) (rig.Spec, error)`; `source.Registry`
  struct; `source.NewRegistry() *Registry`;
  `(*Registry).Register(Source) error`;
  `(*Registry).Get(string) (Source, bool)`; `(*Registry).Names() []string`
  returning sorted names.

No concrete source is implemented here. The text source needs a catalog to
constrain generation against, and no catalog exists until exports arrive. This
task establishes the seam so that adding one later touches nothing else.

- [ ] **Step 1: Write the failing test**

Create `pkg/source/registry_public_test.go`:

```go
package source_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/rig"
	"github.com/retr0h/helix-core/pkg/source"
)

type stubSource struct {
	name string
	spec rig.Spec
	err  error
}

func (s *stubSource) Name() string { return s.name }

func (s *stubSource) Resolve(context.Context, source.Request) (rig.Spec, error) {
	return s.spec, s.err
}

type RegistryPublicTestSuite struct {
	suite.Suite
}

func (s *RegistryPublicTestSuite) TestRegisterThenGet() {
	r := source.NewRegistry()
	src := &stubSource{name: "text"}

	s.Require().NoError(r.Register(src))

	got, ok := r.Get("text")
	s.Require().True(ok)
	s.Require().Same(src, got)
}

func (s *RegistryPublicTestSuite) TestGetReportsFalseForUnknownName() {
	r := source.NewRegistry()

	_, ok := r.Get("nope")
	s.Require().False(ok)
}

func (s *RegistryPublicTestSuite) TestRegisterRejectsADuplicateName() {
	r := source.NewRegistry()

	s.Require().NoError(r.Register(&stubSource{name: "text"}))
	s.Require().Error(r.Register(&stubSource{name: "text"}))
}

func (s *RegistryPublicTestSuite) TestRegisterRejectsAnEmptyName() {
	r := source.NewRegistry()

	s.Require().Error(r.Register(&stubSource{name: ""}))
}

func (s *RegistryPublicTestSuite) TestRegisterRejectsANilSource() {
	r := source.NewRegistry()

	s.Require().Error(r.Register(nil))
}

func (s *RegistryPublicTestSuite) TestNamesAreSorted() {
	r := source.NewRegistry()

	for _, n := range []string{"youtube", "text", "audio"} {
		s.Require().NoError(r.Register(&stubSource{name: n}))
	}

	s.Require().Equal([]string{"audio", "text", "youtube"}, r.Names())
}

func (s *RegistryPublicTestSuite) TestNamesIsEmptyForANewRegistry() {
	s.Require().Empty(source.NewRegistry().Names())
}

func TestRegistryPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryPublicTestSuite))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd helix-core && mise exec -- go test ./pkg/source/... -v` Expected: FAIL
— package `source` does not exist.

- [ ] **Step 3: Write the interface**

Create `pkg/source/source.go`:

```go
// Package source turns a request for a sound into a rig. Every input path —
// a description, an artist name, a recording — is a Source, and a Source is
// the only place in helix-core where I/O belongs.
package source

import (
	"context"

	"github.com/retr0h/helix-core/pkg/rig"
)

// Request is what a caller asks for. Sources ignore fields they cannot use.
type Request struct {
	// Text is the description of the wanted sound, as written.
	Text string
	// Device names the target hardware, so a source can pick a catalog.
	Device string
}

// Source resolves a request to a rig. Implementations perform whatever I/O
// they need; everything downstream sees only the resulting rig.Spec.
type Source interface {
	// Name identifies the source in a registry and in a rig's provenance.
	Name() string
	// Resolve returns the rig for req, or helixerr.ErrNoMatch when the source
	// recognises nothing in it.
	Resolve(ctx context.Context, req Request) (rig.Spec, error)
}
```

- [ ] **Step 4: Write the registry**

Create `pkg/source/registry.go`:

```go
package source

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Registry holds the sources an application can resolve against, keyed by name.
// Its zero value is not usable; call NewRegistry.
type Registry struct {
	sources map[string]Source
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{sources: make(map[string]Source)}
}

// Register adds s under its own name. It reports an error for a nil source, an
// empty name, or a name already registered — silently replacing a source would
// make the resolution order depend on initialisation order.
func (r *Registry) Register(s Source) error {
	if s == nil {
		return errors.New("registering a nil source")
	}

	name := s.Name()
	if name == "" {
		return errors.New("registering a source with an empty name")
	}

	if _, exists := r.sources[name]; exists {
		return fmt.Errorf("source %q is already registered", name)
	}

	r.sources[name] = s

	return nil
}

// Get returns the source registered under name and whether one exists.
func (r *Registry) Get(name string) (Source, bool) {
	s, ok := r.sources[name]

	return s, ok
}

// Names returns every registered name in sorted order.
func (r *Registry) Names() []string {
	return slices.Sorted(maps.Keys(r.sources))
}
```

- [ ] **Step 5: Run the test, run the gate, stage**

```bash
cd helix-core
mise exec -- go test ./pkg/source/... -v
mise exec -- just ready
git add pkg/source
git status --short
```

Suggested subject: `feat(source): add Source interface and registry`

______________________________________________________________________

### Task 10: helixctl validate command

**Files:**

- Create: `helixctl/internal/validate/validate.go`
- Create: `helixctl/internal/validate/validate_public_test.go`
- Create: `helixctl/internal/validate/testdata/catalog.json`
- Create: `helixctl/internal/validate/testdata/rig_ok.json`
- Create: `helixctl/internal/validate/testdata/rig_bad.json`
- Create: `helixctl/cmd/root.go`
- Create: `helixctl/cmd/validate.go`
- Modify: `helixctl/main.go`
- Modify: `helixctl/go.mod`

**Interfaces:**

- Consumes: `catalog.Load`, `rig.Spec`, `rig.Validate`, `rig.HXStompLimits`.
- Produces: `validate.Options{CatalogPath, RigPath string}`;
  `validate.Run(io.Writer, Options) error`.

`helixctl/.coverignore` excludes `/cmd/` and `main.go` but **not** `internal/`.
Cobra wiring therefore goes in `cmd/` where the coverage gate does not reach it,
and everything with behaviour goes in `internal/validate/` where it is tested to
100%. This is the same split the boundary rules describe: the CLI holds no logic
worth testing.

- [ ] **Step 1: Wire the module dependency**

```bash
cd helixctl
mise exec -- go mod edit -require=github.com/retr0h/helix-core@v0.0.0
mise exec -- go mod edit -replace=github.com/retr0h/helix-core=../helix-core
mise exec -- go get github.com/spf13/cobra@latest
mise exec -- go get github.com/stretchr/testify@latest
mise exec -- go mod tidy
```

The `replace` directive is local scaffolding. It comes out when `helix-core` is
published and must not survive extraction.

- [ ] **Step 2: Write the fixtures**

Create `internal/validate/testdata/catalog.json`:

```json
{
  "device": "HX Stomp",
  "device_id": 2162689,
  "schema_version": 6,
  "modeldata_version": 30,
  "blocks": {
    "HD2_AmpTest": {
      "id": "HD2_AmpTest",
      "name": "Test Amp",
      "category": "amp",
      "stereo": false,
      "dsp": { "mono": 0.30, "stereo": 0.55, "prov": "measured" },
      "prov": "observed",
      "params": {
        "Gain": {
          "key": "Gain", "label": "Drive", "type": "float",
          "min": 0.0, "max": 1.0, "default": 0.5, "unit": ""
        }
      }
    }
  }
}
```

Create `internal/validate/testdata/rig_ok.json`:

```json
{
  "name": "Test Rig",
  "origin": "curated",
  "blocks": [
    { "model": "HD2_AmpTest", "params": { "Gain": 0.5 }, "dsp": 0, "pos": 0, "enabled": true }
  ]
}
```

Create `internal/validate/testdata/rig_bad.json`:

```json
{
  "name": "Broken Rig",
  "origin": "curated",
  "blocks": [
    { "model": "HD2_DoesNotExist", "params": {}, "dsp": 0, "pos": 0, "enabled": true }
  ]
}
```

- [ ] **Step 3: Write the failing test**

Create `internal/validate/validate_public_test.go`:

```go
package validate_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/helix-core/pkg/helixerr"
	"github.com/retr0h/helixctl/internal/validate"
)

type ValidatePublicTestSuite struct {
	suite.Suite
}

func (s *ValidatePublicTestSuite) path(name string) string {
	return filepath.Join("testdata", name)
}

func (s *ValidatePublicTestSuite) TestReportsAValidRig() {
	var out bytes.Buffer

	err := validate.Run(&out, validate.Options{
		CatalogPath: s.path("catalog.json"),
		RigPath:     s.path("rig_ok.json"),
	})

	s.Require().NoError(err)
	s.Require().Contains(out.String(), "Test Rig")
	s.Require().Contains(out.String(), "valid")
}

func (s *ValidatePublicTestSuite) TestReportsAnInvalidRig() {
	var out bytes.Buffer

	err := validate.Run(&out, validate.Options{
		CatalogPath: s.path("catalog.json"),
		RigPath:     s.path("rig_bad.json"),
	})

	s.Require().ErrorIs(err, helixerr.ErrUnknownBlock)
}

func (s *ValidatePublicTestSuite) TestReportsAMissingCatalog() {
	var out bytes.Buffer

	err := validate.Run(&out, validate.Options{
		CatalogPath: s.path("nope.json"),
		RigPath:     s.path("rig_ok.json"),
	})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "catalog")
}

func (s *ValidatePublicTestSuite) TestReportsAMissingRig() {
	var out bytes.Buffer

	err := validate.Run(&out, validate.Options{
		CatalogPath: s.path("catalog.json"),
		RigPath:     s.path("nope.json"),
	})

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "rig")
}

func (s *ValidatePublicTestSuite) TestReportsAMalformedRig() {
	var out bytes.Buffer

	err := validate.Run(&out, validate.Options{
		CatalogPath: s.path("catalog.json"),
		RigPath:     s.path("catalog.json"), // valid JSON, wrong shape
	})

	s.Require().Error(err)
}

func TestValidatePublicTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatePublicTestSuite))
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `cd helixctl && mise exec -- go test ./internal/... -v` Expected: FAIL —
package `validate` does not exist.

- [ ] **Step 5: Write the implementation**

Create `internal/validate/validate.go`:

```go
// Package validate checks a rig against a catalog and reports the result. It
// holds the command's behaviour; cmd/ holds only the flag wiring.
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/retr0h/helix-core/pkg/catalog"
	"github.com/retr0h/helix-core/pkg/rig"
)

// Options are the inputs the validate command takes.
type Options struct {
	// CatalogPath is the catalog JSON to validate against.
	CatalogPath string
	// RigPath is the rig JSON to validate.
	RigPath string
}

// Run validates the rig at opts.RigPath against the catalog at
// opts.CatalogPath, writing a human-readable result to w. It returns the
// validation error unchanged so a caller can match it with errors.Is.
func Run(w io.Writer, opts Options) error {
	cf, err := os.Open(opts.CatalogPath)
	if err != nil {
		return fmt.Errorf("opening catalog: %w", err)
	}
	defer func() { _ = cf.Close() }()

	cat, err := catalog.Load(cf)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	rf, err := os.Open(opts.RigPath)
	if err != nil {
		return fmt.Errorf("opening rig: %w", err)
	}
	defer func() { _ = rf.Close() }()

	var spec rig.Spec
	if err := json.NewDecoder(rf).Decode(&spec); err != nil {
		return fmt.Errorf("decoding rig: %w", err)
	}

	if err := rig.Validate(cat, spec, rig.HXStompLimits()); err != nil {
		return err
	}

	fmt.Fprintf(w, "%s: valid — %d block(s) against %s\n",
		spec.Name, len(spec.Blocks), cat.Device)

	return nil
}
```

Note: `rig.Validate` takes a `rig.BlockLookup`. `*catalog.Catalog` satisfies it
through its `Block` method, so `cat` passes directly.

`TestReportsAMalformedRig` passes `catalog.json` as the rig. It decodes into a
`rig.Spec` with no blocks, which `ValidateTopology` rejects as "rig has no
blocks" — so the error arrives from validation rather than decoding. Both are
errors; the test asserts only that one occurs.

- [ ] **Step 6: Write the cobra wiring**

Create `cmd/root.go`:

```go
// Package cmd wires flags to behaviour. It holds no logic worth testing and is
// excluded from the coverage gate by .coverignore.
package cmd

import "github.com/spf13/cobra"

// NewRootCmd returns the helixctl root command with every subcommand attached.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "helixctl",
		Short:         "Generate and inspect Line 6 Helix presets",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(newValidateCmd())

	return root
}
```

Create `cmd/validate.go`:

```go
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/retr0h/helixctl/internal/validate"
)

func newValidateCmd() *cobra.Command {
	var opts validate.Options

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check a rig against a catalog",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validate.Run(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.CatalogPath, "catalog", "", "path to the catalog JSON")
	cmd.Flags().StringVar(&opts.RigPath, "rig", "", "path to the rig JSON")
	_ = cmd.MarkFlagRequired("catalog")
	_ = cmd.MarkFlagRequired("rig")

	return cmd
}
```

Replace `main.go`:

```go
// Command helixctl generates and inspects Line 6 Helix presets.
package main

import (
	"os"

	"github.com/retr0h/helixctl/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `cd helixctl && mise exec -- go test ./internal/... -v` Expected: PASS.

- [ ] **Step 8: Verify end to end by hand**

```bash
cd helixctl
mise exec -- go build -o helixctl .
./helixctl validate \
  --catalog internal/validate/testdata/catalog.json \
  --rig internal/validate/testdata/rig_ok.json
```

Expected: `Test Rig: valid — 1 block(s) against HX Stomp`

```bash
./helixctl validate \
  --catalog internal/validate/testdata/catalog.json \
  --rig internal/validate/testdata/rig_bad.json
echo "exit: $?"
```

Expected: an error naming `HD2_DoesNotExist`, exit 1.

- [ ] **Step 9: Run the full gate, stage**

```bash
cd helixctl
mise exec -- just ready
git add cmd internal main.go go.mod go.sum
git status --short
```

Suggested subject: `feat(cmd): add validate command`

______________________________________________________________________

## What this plan does not build, and why

| Deliverable                       | Blocked on                                                                  |
| --------------------------------- | --------------------------------------------------------------------------- |
| Envelope validation (layer 5)     | The real `device`, `version` and `modeldata_version` integers               |
| Synth writer (`RigSpec` → `.hlx`) | Whether values are normalised; whether key order matters; snapshot encoding |
| Catalog extractor                 | Real exports to diff                                                        |
| Text source (recipes + Claude)    | A catalog to constrain generation against                                   |
| Hardware verification             | An HX Stomp and files to load on it                                         |

All five need `.hlx` files exported from the target device. Building any of them
against assumptions would produce code that compiles, passes its own tests, and
emits presets the hardware rejects — the most expensive kind of wrong, because
the tests would say it works.

They get a second plan once exports exist.

## Verification

At the end of every task:

```bash
mise exec -- just ready     # format, lint, vet, test, 100% coverage
```

Three distinct claims, not interchangeable. Say which one you have:

1. the rig validates against the catalog
2. HX Edit imports the generated file
3. the hardware loads it and it sounds as the rig described

This plan reaches only (1). Nothing in it can support a claim of (2) or (3).
