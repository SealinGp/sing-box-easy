package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const pkgRoot = "github.com/SealinGp/sing-box-easy/app/pkg/"

// layerRule forbids a set of imports to every non-test file in one directory
// (dir) or in a whole subtree (dir + "/...").
type layerRule struct {
	dir       string
	forbidden []string // import paths relative to pkgRoot; a trailing "/..." forbids the subtree
	why       string
}

// The sing-box tree is layered bottom-up:
//
//	singbox/core                 the binary: version, check, capabilities
//	singbox/clashapi             the running process: its Clash-compatible API
//	singbox/config               the config document store (Manager)
//	singbox/config/{sections,outbounds}   editors, siblings of each other
//	singbox                      the process controller, which consumes config
//
// Nothing points upward, and the two editor families never reach into each
// other — subscription depends on outbounds, and must not drag the section
// editors in with it.
var singBoxLayers = []layerRule{
	{
		dir:       "singbox/core/...",
		forbidden: []string{"..."},
		why:       "core is the leaf every layer builds on",
	},
	{
		dir:       "singbox/clashapi/...",
		forbidden: []string{"..."},
		why:       "clashapi is the leaf boundary to the running process, as core is to the binary",
	},
	{
		dir:       "singbox/config/...",
		forbidden: []string{"singbox/clashapi/..."},
		why:       "editing the config must not depend on a running sing-box",
	},
	{
		dir:       "singbox/config",
		forbidden: []string{"singbox/config/sections/...", "singbox/config/outbounds/...", "subscription/..."},
		why:       "the document store must not know which editors sit on top of it",
	},
	{
		dir:       "singbox/config/...",
		forbidden: []string{"singbox"},
		why:       "config sits below the process controller",
	},
	{
		dir:       "singbox/config/sections/...",
		forbidden: []string{"singbox/config/outbounds/..."},
		why:       "the section editors and the outbound editor are siblings",
	},
	{
		dir:       "singbox/config/outbounds/...",
		forbidden: []string{"singbox/config/sections/...", "subscription/..."},
		why:       "outbounds is imported BY subscription and must stay independent of the section editors",
	},
	{
		dir:       "singbox/config/outbounds/nodegroup/...",
		forbidden: []string{"singbox/config/outbounds", "singbox/config/outbounds/rules/..."},
		why:       "nodegroup is a leaf both outbounds and rules import; an edge back would be a cycle",
	},
	{
		dir:       "singbox/config/outbounds/nodetag/...",
		forbidden: []string{"singbox/config/outbounds", "singbox/config/outbounds/rules/..."},
		why:       "nodetag is a leaf both outbounds and rules import; an edge back would be a cycle",
	},
}

func TestSingBoxLayering(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	pkgDir := filepath.Join(filepath.Dir(filepath.Dir(file)), "pkg")
	err := filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		relDir, _ := filepath.Rel(pkgDir, filepath.Dir(path))
		relDir = filepath.ToSlash(relDir)
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			value, _ := strconv.Unquote(imp.Path.Value)
			if !strings.HasPrefix(value, pkgRoot) {
				continue
			}
			target := strings.TrimPrefix(value, pkgRoot)
			for _, rule := range singBoxLayers {
				if !matches(rule.dir, relDir) {
					continue
				}
				for _, forbidden := range rule.forbidden {
					if matches(forbidden, target) {
						t.Errorf("%s imports %s: %s", relDir, target, rule.why)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// matches reports whether path is pattern, or lies under it when the pattern
// ends in "/..."; the bare pattern "..." matches everything.
func matches(pattern, path string) bool {
	if pattern == "..." {
		return true
	}
	if base, ok := strings.CutSuffix(pattern, "/..."); ok {
		return path == base || strings.HasPrefix(path, base+"/")
	}
	return path == pattern
}
