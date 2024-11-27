package astinfo

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// 针对一个真正的go文件，这个类时临时对象，仅仅是解析过程中存在
type GoFile struct {
	file    *ast.File
	pkg     *Package
	path    string            //一个go文件的全路径
	Imports map[string]string //本go文件import值
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
			// type interface, type struct
			switch genDecl.Tok {
			case token.TYPE:
				{
					if len(genDecl.Specs) > 1 {
						log.Fatalf("解析结构体时，发现多个结构，代码功能不全 %s 下标%d\n", goFile.path, i)
					}
					goFile.parseType(genDecl)
				}
			case token.VAR:
				{
					//解析package中的全局变量
					goFile.parseVariable(genDecl)
				}
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

// 有些第三方的modeName不是最后package的最后一位，所以此处可能找不到；先退出程序，后续看看怎么处理
// 返回go文件头中import的modeName对应的全路径
func (gofile *GoFile) getImportPath(modeName string, info string) string {
	pkgPath := gofile.Imports[modeName]
	if len(pkgPath) == 0 {
		fmt.Printf("failed to find the fullPath of package %s in %s, please add alias %s to your import part to avoid this\n", modeName, info, modeName)
		os.Exit(1)
	}
	return pkgPath
}

// 解析go文件的Import字段，如果有modeName直接使用，否则用pathValue的文件名；
// 注意此处可能有错误，因为有些package的模块名不是路径的最后一位；
// 此时只能通过解析原package文件才能解决；否则后面getImportPath就找不到了
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
		// pkg := goFile.pkg.Project.getPackage(pathValue, true)
		// 此处是第三方package，也可能是本项目的尚未被解析的工程，其modeName为空，先补一个；
		// 主要是为了解决package的ModeName不是其path的最后的baseName的情况
		// if len(pkg.modName) == 0 {
		// 	pkg.modName = name
		// }
		goFile.Imports[name] = pathValue
	}
}

// 解析package中的全局变量
func (goFile *GoFile) parseVariable(genDecl *ast.GenDecl) {
	if fieldPair, ok := genDecl.Specs[0].(*ast.ValueSpec); ok {
		name := fieldPair.Names[0].Name
		var field = Field{
			name: name,
		}
		field.parseType(fieldPair.Type, goFile)
		intface := field.findInterface()
		if intface != nil {
			if intface.config != nil {
				field.typeName = intface.Name
				field.pkg = goFile.pkg
				goFile.pkg.Project.addInitRpcClientFuns(&field)
			}
		}
	}
}
func (goFile *GoFile) parseType(genDecl *ast.GenDecl) {
	typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
	// 仅关注结构体，暂时不考虑接口
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType:
		interfaceType := typeSpec.Type.(*ast.InterfaceType)
		itface := goFile.pkg.getInterface(typeSpec.Name.Name, true)
		itface.parseComment(genDecl.Doc)
		itface.Parse(interfaceType, goFile)
		fmt.Printf("interface %s\n", typeSpec.Name.Name)
	case *ast.StructType:
		structType := typeSpec.Type.(*ast.StructType)
		class := goFile.pkg.getStruct(typeSpec.Name.Name, true)
		class.structFound = true
		parseComment(genDecl.Doc, &class.comment)
		if class.comment.serverType != NOUSAGE {
			goFile.pkg.Project.addServer(class.comment.groupName)
		}
		class.parse(structType, goFile)
	}
}

// 解析对象的方法, 解析完成后自动塞到receiver对象中去
func (goFile *GoFile) parseMethod(method *ast.FuncDecl) {
	method1 := createMethod(method, goFile)
	method1.Parse()
}
func (goFile *GoFile) parseFunction(funcDecl *ast.FuncDecl) {
	function1 := createFunction(funcDecl, goFile, &goFile.pkg.FunctionManager)
	function1.Parse()
}
