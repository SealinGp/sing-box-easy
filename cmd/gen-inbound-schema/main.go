// Command gen-inbound-schema reflects over sing-box's own inbound option
// structs and emits the field inventory the frontend forms are typed against.
//
// WHY THIS EXISTS
// ───────────────
// The inbound modal used to hand-write a `v-if` block per type, and
// frontend/src/types/inbound.ts hand-transcribed the same option structs into
// TypeScript a second time. Both drifted: twelve of sixteen types rendered no
// type-specific fields at all, and "anytls" — registered on the backend since
// 1.12 — never appeared in either the TS union or the type dropdown.
//
// The authoritative field list already exists, exactly and version-correctly,
// in the structs app/pkg/config/registry.go constructs. This reads those
// structs and writes them out, so a sing-box upgrade that renames or removes a
// field becomes a `vue-tsc` failure instead of a form that silently stops
// editing something.
//
// HOW DEPRECATION IS DETECTED
// ───────────────────────────
// Two sources, because neither is complete on its own:
//
//   - Most deprecated fields carry a `// Deprecated:` doc comment in the
//     sing-box source. reflect cannot see comments, so the option package is
//     parsed with go/ast to recover them. This was not optional: a
//     hand-written list got 2 of 8 tun fields wrong on the first attempt,
//     missing inet4_route_exclude_address and inet6_route_exclude_address.
//
//   - Some fields are deprecated only in the documentation, with no marker in
//     the source at all — the whole sniff family moved to route rules in 1.12
//     while option.InboundOptions still declares them plainly. Those are listed
//     in docDeprecatedFields below and must be rechecked on upgrade.
//
// WHAT IT DOES NOT KNOW
// ─────────────────────
// Which fields matter. Ordering, labels and the
// core/typical/advanced tiering are editorial and live in the curation file on
// the frontend; this only supplies which fields exist and what shape they are.
//
// Usage:
//
//	go run ./cmd/gen-inbound-schema
//	go run ./cmd/gen-inbound-schema -o path/to/out.ts
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

const defaultOutput = "frontend/src/schemas/inboundInventory.generated.ts"

// singBoxModule is the dependency whose version is stamped into the generated
// header, so a reader can tell which sing-box the inventory describes.
const singBoxModule = "github.com/sagernet/sing-box"

// Field kinds the frontend can render. Anything this generator cannot classify
// degrades to "json" — an explicit "edit this as raw JSON" signal rather than a
// guess that renders the wrong control.
const (
	kindAddress  = "address"
	kindBoolean  = "boolean"
	kindCIDR     = "cidr"
	kindDuration = "duration"
	kindJSON     = "json"
	kindList     = "list"
	kindNumber   = "number"
	kindObject   = "object"
	kindString   = "string"
)

// namedKinds maps sing-box's wrapper types to a UI kind. These are checked
// before reflect.Kind because the underlying kind is misleading: a Duration is
// an int64 on the wire but a string ("5m") in JSON, and an Addr is a struct
// that must render as a single address input, not an object editor.
// Both the badoption wrappers and the stdlib types they wrap appear in the
// option structs — ListenOptions uses *badoption.Addr for `listen` while
// TunInboundOptions uses a bare netip.Prefix for `address`. Missing the stdlib
// half classified every tun address field as a generic object.
var namedKinds = map[string]string{
	"badoption.Duration":      kindDuration,
	"option.UDPTimeoutCompat": kindDuration,
	"badoption.Addr":          kindAddress,
	"netip.Addr":              kindAddress,
	"badoption.Prefix":        kindCIDR,
	"badoption.Prefixable":    kindCIDR,
	"netip.Prefix":            kindCIDR,
	"option.FwMark":           kindNumber,
	"option.DomainStrategy":   kindString,
	"option.NetworkList":      kindString,
}

// docDeprecatedFields covers fields the sing-box DOCS retire but the source
// still declares without a marker, so the go/ast pass cannot find them. Keyed
// by JSON name; these all live in the shared ListenOptions / InboundOptions and
// therefore apply to every listening inbound.
//
// Deprecated fields stay in the inventory on purpose: existing config.json
// files carry them, and a form that dropped them would silently discard a live
// setting on the next save. The frontend files them under the advanced tier
// with a warning instead.
//
// Recheck against the Listen Fields docs when bumping sing-box.
var docDeprecatedFields = map[string]bool{
	// Superseded by route rules with `action: sniff` in 1.12.
	"sniff":                        true,
	"sniff_override_destination":   true,
	"sniff_timeout":                true,
	"domain_strategy":              true,
	"udp_disable_domain_unmapping": true,
}

type field struct {
	Name       string
	Kind       string
	Item       string // element kind, for kindList
	Deprecated bool
	depth      int // embedding depth; shallower wins on a name collision

	// Where this field came from in Go, so the go/ast deprecation pass can be
	// matched up with it. Not emitted.
	goStruct string
	goField  string
}

func main() {
	out := flag.String("o", "", "output path (default "+defaultOutput+", relative to repo root)")
	flag.Parse()

	target := *out
	if target == "" {
		root, err := repoRoot()
		if err != nil {
			fatal(err)
		}
		target = filepath.Join(root, defaultOutput)
	}

	source, err := render()
	if err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(target, source, 0o644); err != nil {
		fatal(err)
	}

	fmt.Fprintf(os.Stderr, "wrote %s (%d inbound types)\n", target, len(config.InboundTypes))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen-inbound-schema:", err)
	os.Exit(1)
}

// repoRoot walks up from the working directory to the go.mod, so the generator
// produces the same output regardless of where `go run` was invoked from.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}

func render() ([]byte, error) {
	registry := &config.Registry{}

	deprecations, err := scanDeprecations()
	if err != nil {
		return nil, err
	}

	type entry struct {
		Type   string
		Fields []field
	}
	entries := make([]entry, 0, len(config.InboundTypes))

	for _, inboundType := range config.InboundTypes {
		options, ok := registry.CreateInboundOptions(inboundType)
		if !ok {
			// config.TestInboundTypesAreRegistered guards this, so reaching it
			// means the generator ran against an inconsistent tree.
			return nil, fmt.Errorf("inbound type %q is listed in config.InboundTypes but not registered", inboundType)
		}

		fields := collect(reflect.TypeOf(options))
		if len(fields) == 0 {
			return nil, fmt.Errorf("inbound type %q reflected to zero fields", inboundType)
		}
		markDeprecated(fields, deprecations)
		entries = append(entries, entry{Type: inboundType, Fields: fields})
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, `// Code generated by cmd/gen-inbound-schema. DO NOT EDIT.
//
// Field inventory reflected from %s %s — the same option structs
// app/pkg/config/registry.go feeds to sing-box's own config parser, so this is
// what the running binary actually accepts.
//
// Regenerate with:  go run ./cmd/gen-inbound-schema
//
// This file says which fields EXIST and what shape they are. It deliberately
// says nothing about which ones matter, what they are called in the UI, or what
// order to show them in — that is editorial and lives in inboundFields.ts,
// which is type-checked against the keys below.

export type InboundFieldKind =
  | '%s'
  | '%s'
  | '%s'
  | '%s'
  | '%s'
  | '%s'
  | '%s'
  | '%s'
  | '%s'

export interface InboundFieldInfo {
  readonly kind: InboundFieldKind
  /** Element kind, for kind: 'list'. */
  readonly item?: InboundFieldKind
  /**
   * Still accepted by sing-box, but documented as deprecated. Kept so an
   * existing config that uses the field can still be edited rather than having
   * it silently dropped on save.
   */
  readonly deprecated?: true
}

export const INBOUND_INVENTORY = {
`,
		singBoxModule, singBoxVersion(),
		kindAddress, kindBoolean, kindCIDR, kindDuration, kindJSON,
		kindList, kindNumber, kindObject, kindString,
	)

	for _, e := range entries {
		fmt.Fprintf(&b, "  %s: {\n", tsKey(e.Type))
		for _, f := range e.Fields {
			fmt.Fprintf(&b, "    %s: { kind: '%s'", tsKey(f.Name), f.Kind)
			if f.Item != "" {
				fmt.Fprintf(&b, ", item: '%s'", f.Item)
			}
			if f.Deprecated {
				b.WriteString(", deprecated: true")
			}
			b.WriteString(" },\n")
		}
		b.WriteString("  },\n")
	}

	b.WriteString(`} as const satisfies Record<string, Record<string, InboundFieldInfo>>

/** Every inbound type the backend registry can construct. */
export type InboundTypeName = keyof typeof INBOUND_INVENTORY

/**
 * The field keys valid for one inbound type. Curation in inboundFields.ts is
 * keyed by this, so naming a field sing-box does not have — or one it dropped
 * in an upgrade — is a compile error rather than an input that never binds.
 */
export type InboundFieldKey<T extends InboundTypeName> = Extract<
  keyof (typeof INBOUND_INVENTORY)[T],
  string
>

export const INBOUND_TYPE_NAMES = Object.keys(INBOUND_INVENTORY) as InboundTypeName[]
`)

	return b.Bytes(), nil
}

// collect flattens a struct's JSON fields, following embedded structs the way
// encoding/json does.
func collect(t reflect.Type) []field {
	var found []field

	var walk func(reflect.Type, int)
	walk = func(t reflect.Type, depth int) {
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue // unexported
			}

			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "-" {
				continue
			}

			// An embedded struct with no JSON name is inlined by the encoder —
			// this is how ListenOptions and InboundOptions contribute `listen`,
			// `sniff`, `detour` and friends to every listening inbound.
			if f.Anonymous && name == "" {
				walk(f.Type, depth+1)
				continue
			}

			if name == "" {
				name = f.Name
			}

			kind, item := classify(f.Type)
			found = append(found, field{
				Name:     name,
				Kind:     kind,
				Item:     item,
				depth:    depth,
				goStruct: t.Name(),
				goField:  f.Name,
			})
		}
	}
	walk(t, 0)

	return dedupe(found)
}

// dedupe resolves name collisions between an outer field and one promoted from
// an embedded struct. Go's encoder gives the shallower field the name, so the
// generated inventory must agree or the form would bind to a key the backend
// never reads.
func dedupe(in []field) []field {
	best := make(map[string]field, len(in))
	for _, f := range in {
		if prev, ok := best[f.Name]; ok && prev.depth <= f.depth {
			continue
		}
		best[f.Name] = f
	}

	out := make([]field, 0, len(best))
	for _, f := range best {
		out = append(out, f)
	}
	// Alphabetical, so regenerating after an unrelated sing-box change produces
	// a diff containing only the fields that actually changed. Presentation
	// order is the curation file's job.
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// classify maps a Go type to the control the frontend should render.
func classify(t reflect.Type) (kind string, item string) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if named, ok := namedKinds[typeName(t)]; ok {
		return named, ""
	}

	switch t.Kind() {
	case reflect.Bool:
		return kindBoolean, ""
	case reflect.String:
		return kindString, ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return kindNumber, ""
	case reflect.Slice, reflect.Array:
		// badoption.Listable[T] is a plain []T underneath, so this covers both
		// real slices and the scalar-or-array wrapper.
		elem, _ := classify(t.Elem())
		return kindList, elem
	case reflect.Struct, reflect.Map:
		return kindObject, ""
	default:
		return kindJSON, ""
	}
}

func typeName(t reflect.Type) string {
	pkg := t.PkgPath()
	if pkg == "" || t.Name() == "" {
		return ""
	}
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 {
		pkg = pkg[idx+1:]
	}
	return pkg + "." + t.Name()
}

// markDeprecated flags fields retired either in the sing-box source (found by
// parsing its doc comments) or in the docs alone (docDeprecatedFields).
func markDeprecated(fields []field, fromSource deprecationIndex) {
	for i := range fields {
		if docDeprecatedFields[fields[i].Name] || fromSource.has(fields[i].goStruct, fields[i].goField) {
			fields[i].Deprecated = true
		}
	}
}

// deprecationIndex maps a Go struct name to the fields carrying a
// `// Deprecated:` doc comment.
type deprecationIndex map[string]map[string]bool

func (d deprecationIndex) has(structName, fieldName string) bool {
	return d[structName][fieldName]
}

func (d deprecationIndex) add(structName, fieldName string) {
	if d[structName] == nil {
		d[structName] = map[string]bool{}
	}
	d[structName][fieldName] = true
}

// scanDeprecations parses the sing-box option package and records every struct
// field whose doc comment starts a `Deprecated:` paragraph — the convention Go
// tooling itself uses.
//
// reflect deliberately discards comments, so this is the only way to recover
// what upstream has already marked. Doing it by hand was tried and was wrong.
func scanDeprecations() (deprecationIndex, error) {
	dir, err := optionPackageDir()
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", dir, err)
	}

	index := deprecationIndex{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				structType, ok := spec.Type.(*ast.StructType)
				if !ok || structType.Fields == nil {
					return true
				}

				for _, f := range structType.Fields.List {
					if f.Doc == nil || !isDeprecated(f.Doc.Text()) {
						continue
					}
					for _, name := range f.Names {
						index.add(spec.Name.Name, name.Name)
					}
				}
				return true
			})
		}
	}

	if len(index) == 0 {
		// sing-box has always had deprecated option fields. An empty index means
		// the parse silently looked at the wrong place, which would ship an
		// inventory that marks nothing — fail instead.
		return nil, fmt.Errorf("no deprecated fields found in %s; the package layout probably changed", dir)
	}
	return index, nil
}

// isDeprecated follows the Go convention: a paragraph beginning "Deprecated:".
// Checking the line start rather than substring-matching avoids flagging a
// comment that merely mentions the word in passing.
func isDeprecated(doc string) bool {
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Deprecated:") {
			return true
		}
	}
	return false
}

// optionPackageDir asks the go tool where the sing-box module was unpacked.
// `go list -m` is used rather than go/build because the dependency only exists
// in the module cache, which GOPATH-style resolution does not find.
func optionPackageDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", singBoxModule).Output()
	if err != nil {
		return "", fmt.Errorf("locating %s: %w", singBoxModule, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("%s has no local directory; run `go mod download`", singBoxModule)
	}
	return filepath.Join(dir, "option"), nil
}

// tsKey quotes an object key only when it is not a bare JS identifier. Every
// sing-box field name is snake_case today, but quoting defensively keeps the
// output valid if one ever contains a dash.
func tsKey(name string) string {
	bare := name != ""
	for i, r := range name {
		isAlpha := r == '_' || r == '$' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isAlpha && !(isDigit && i > 0) {
			bare = false
			break
		}
	}
	if bare {
		return name
	}
	return "'" + strings.ReplaceAll(name, "'", "\\'") + "'"
}

// singBoxVersion reads the linked sing-box version from the build info, so the
// generated header cannot claim a version the code was not built against.
func singBoxVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown version)"
	}
	for _, dep := range info.Deps {
		if dep.Path == singBoxModule {
			if dep.Replace != nil {
				return dep.Replace.Version
			}
			return dep.Version
		}
	}
	return "(unknown version)"
}
