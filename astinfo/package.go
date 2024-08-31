package astinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
)

type Package struct {
	Struct  []*Struct
	ModInfo *Import
	Project *Project
}

type Import struct {
	Name string
	Path string
}

func (pkg *Package) Parse(path string) {
	fset := token.NewFileSet()
	// 这里取绝对路径，方便打印出来的语法树可以转跳到编辑器
	packageMap, err := parser.ParseDir(fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Printf("parse %s failed %s", path, err.Error())
		return
	}
	for packName, pack := range packageMap {
		fmt.Print(packName, "\n")
		for filename, f := range pack.Files {
			fmt.Println(filename)
			for i := 0; i < len(f.Decls); i++ {
				if function, ok := f.Decls[i].(*ast.FuncDecl); ok {
					pkg.parseMethod(function)
				}
			}
		}
	}
}
func (pkg *Package) parseMethod(method *ast.FuncDecl) {
	_ = method.Recv
	method1 := Method{}
	method1.InitFromFunc(method)
}
