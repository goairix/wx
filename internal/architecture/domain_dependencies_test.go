package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Legacy constructors can mention the internal concrete adapter for source
// compatibility; domain state must only depend on its callable contract.
func TestDomainStateDoesNotDependOnConcreteExecutor(t *testing.T) {
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		platform, domain := platformDomain(modulePath + "/" + filepath.ToSlash(relative))
		if platform == "" || domain == "" || sharedPlatformPackage(domain) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		aliases := map[string]bool{}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if importPath != modulePath+"/"+platform+"/internal/api" {
				continue
			}
			alias := "api"
			if imported.Name != nil {
				alias = imported.Name.Name
			}
			aliases[alias] = true
		}
		ast.Inspect(file, func(node ast.Node) bool {
			structure, ok := node.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structure.Fields.List {
				fieldType := field.Type
				if pointer, ok := fieldType.(*ast.StarExpr); ok {
					fieldType = pointer.X
				}
				selector, ok := fieldType.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "Client" {
					continue
				}
				qualifier, ok := selector.X.(*ast.Ident)
				if ok && aliases[qualifier.Name] {
					t.Errorf("%s: domain state stores concrete internal/api.Client", path)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
