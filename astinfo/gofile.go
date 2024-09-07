package astinfo

import (
	"go/ast"
	"go/token"
	"log"
	"path/filepath"
	"strings"
)

// 针对一个真正的go文件，这个类时临时对象，仅仅是解析过程中存在
type GoFile struct {
	file    *ast.File
	pkg     *Package
	path    string //一个go文件的全路径
	Imports map[string]string
}

func createGoFile(file *ast.File, pkg *Package, path string) GoFile {
	return GoFile{
		file:    file,
		pkg:     pkg,
		path:    path,
		Imports: make(map[string]string),
	}
}
func (goFile *GoFile) parseFile() {
	goFile.parseImport()
	decls := goFile.file.Decls
	for i := 0; i < len(decls); i++ {
		if genDecl, ok := decls[i].(*ast.GenDecl); ok {
			if genDecl.Tok == token.TYPE {
				if len(genDecl.Specs) > 1 {
					log.Fatalf("解析结构体时，发现多个结构，代码功能不全 %s 下标%d\n", goFile.path, i)
				}

				goFile.parseStruct(genDecl.Specs[0].(*ast.TypeSpec))
			}
		} else if funcDecl, ok := decls[i].(*ast.FuncDecl); ok {
			if funcDecl.Recv == nil {
				goFile.parseFunction(funcDecl)
			} else {
				goFile.parseMethod(funcDecl)
			}
		}
	}
}

func (goFile *GoFile) parseImport() {
	astFile := goFile.file
	for _, importSpec := range astFile.Imports {
		var name string
		pathValue := strings.Trim(importSpec.Path.Value, "\"")
		if importSpec.Name != nil {
			name = importSpec.Name.Name
		} else {
			name = filepath.Base(pathValue)
		}
		goFile.Imports[name] = pathValue
	}
}

func (goFile *GoFile) parseStruct(class *ast.TypeSpec) {
	// 仅关注结构体，暂时不考虑接口
	if structType, ok := class.Type.(*ast.StructType); ok {
		_ = structType
		struct1 := goFile.pkg.getStruct(class.Name.Name, true)
		struct1.structFound = true
	}
}

// 解析对象的方法, 解析完成后自动塞到receiver对象中去
func (goFile *GoFile) parseMethod(method *ast.FuncDecl) {
	method1 := createMethod(method, goFile)
	method1.Parse()
}
func (goFile *GoFile) parseFunction(function *ast.FuncDecl) {

}
