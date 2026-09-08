package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// These checks enforce ownership without making the HTTP framework a test fixture.
func TestModuleDependencies(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(file))
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			value, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(relative, "routes/") {
				for _, forbidden := range []string{"/repo", "/integrations/", "/platform/", "/database", "/internal/", "xorm.io"} {
					if strings.Contains(value, forbidden) {
						t.Errorf("HTTP adapter %s imports implementation %s", relative, value)
					}
				}
			}
			if strings.HasPrefix(relative, "pkg/") && !strings.HasPrefix(relative, "pkg/logger/") && (strings.Contains(value, "cloudwego/hertz") || strings.Contains(value, "/app/routes")) {
				t.Errorf("business module %s imports transport %s", relative, value)
			}
			if strings.Contains(relative, "/feed/parser/") && (value == "net" || value == "net/http" || value == "os" || strings.Contains(value, "/fetch")) {
				t.Errorf("pure parser imports side effects: %s", value)
			}
		}
		if strings.HasPrefix(relative, "routes/") {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				for _, name := range []string{"UpdateOutboundsConfig", "UpdateConfigSection", "UpdateConfigSections", "Exec", "NewSession"} {
					if selector.Sel.Name == name {
						t.Errorf("HTTP adapter %s performs %s", relative, name)
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
