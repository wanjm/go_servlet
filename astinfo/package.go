package astinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

type Package struct {
	Struct    []*Struct
	ModInfo   Import
	Project   *Project
	StructMap map[string]*Struct
}

type Import struct {
	Name string
	Path string
}

func CreatePackage(project *Project, modPath string) *Package {
	return &Package{
		Project:   project,
		StructMap: make(map[string]*Struct),
		ModInfo: Import{
			Path: modPath,
		},
	}
}

// 解析本目录下的所有文件，他们的package 相同
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
			pkg.parseFile(f, filename)
		}
	}
}

func (pkg *Package) parseFile(f *ast.File, filename string) {
	pkg.parseMod(f, filename)
	for i := 0; i < len(f.Decls); i++ {
		if function, ok := f.Decls[i].(*ast.FuncDecl); ok {
			if function.Recv == nil {
				pkg.parseFunction(function)
			} else {
				pkg.parseMethod(function)
			}
		}
	}
}

func (pkg *Package) parseMod(file *ast.File, fileName string) {
	if len(pkg.ModInfo.Name) == 0 {
		pkg.ModInfo.Name = file.Name.Name
	} else {
		// 多个文件的mod应该相同，否则报错
		if pkg.ModInfo.Name != file.Name.Name {
			log.Fatalf("mod of %s is %s which should be %s\n", fileName, file.Name.Name, pkg.ModInfo.Name)
		}
	}
}

// 解析对象的方法
func (pkg *Package) parseMethod(method *ast.FuncDecl) {
	recvType := method.Recv.List[0].Type
	var nameIndent *ast.Ident
	if starExpr, ok := recvType.(*ast.StarExpr); ok {
		nameIndent = starExpr.X.(*ast.Ident)
	} else {
		nameIndent = recvType.(*ast.Ident)
	}
	class := pkg.getStruct(nameIndent.Name, true)
	method1 := Method{Receiver: class}
	method1.InitFromFunc(method)
}
func (pkg *Package) getStruct(name string, create bool) *Struct {
	var class *Struct
	var ok bool
	if class, ok = pkg.StructMap[name]; !ok {
		if create {
			class = &Struct{Package: pkg}
			pkg.StructMap[name] = class
		}
	}
	return class
}

// 解析普通对象
func (pkg *Package) parseFunction(method *ast.FuncDecl) {
}

// 生成代码

func (pkg *Package) GenerateCode() string {
	var sb strings.Builder
	for _, class := range pkg.Struct {
		sb.WriteString(class.GenerateCode())
	}
	return sb.String()
}
