package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type PlasoParser struct {
	Pkgs map[string]*ast.Package
}

func (pparse *PlasoParser) ParseDir(fset *token.FileSet, path string) (first error) {
	list, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, d := range list {
		if d.IsDir() {
			pparse.ParseDir(fset, filepath.Join(path, d.Name()))
			continue
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			continue
		}
		// if filter != nil {
		// 	info, err := d.Info()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if !filter(info) {
		// 		continue
		// 	}
		// }
		filename := filepath.Join(path, d.Name())
		if src, err := parser.ParseFile(fset, filename, nil, parser.ParseComments|parser.AllErrors); err == nil {
			name := src.Name.Name
			pkg, found := pparse.Pkgs[name]
			if !found {
				pkg = &ast.Package{
					Name:  name,
					Files: make(map[string]*ast.File),
				}
				pparse.Pkgs[name] = pkg
			}
			pkg.Files[filename] = src
		} else if first == nil {
			first = err
		}
	}

	return
}
