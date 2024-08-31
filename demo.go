package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
)

func main() {
	fset := token.NewFileSet()
	// 这里取绝对路径，方便打印出来的语法树可以转跳到编辑器
	path, _ := filepath.Abs("../server/sys/biz")
	packageMap, err := parser.ParseDir(fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		log.Println(err)
		return
	}
	for packName, pack := range packageMap {
		fmt.Print(packName, "\n")
		for filename, f := range pack.Files {
			fmt.Println(filename)
			for i := 0; i < len(f.Decls); i++ {
				if function, ok := f.Decls[i].(*ast.FuncDecl); ok {
					log.Println(function.Name.Name)
				}
			}
		}

	}
	// 打印语法树

}

func dealFunction(f *ast.FuncDecl) {
	fmt.Println(f.Name.Name)
}
