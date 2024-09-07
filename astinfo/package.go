package astinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

type PackageInfo struct {
	modName string //本package的mode name；基本没有什么用，本程序不检查；
	modPath string //本package的mode path全路径
}

type Package struct {
	// Struct    []*Struct
	// ModInfo   Import
	PackageInfo
	Project   *Project
	StructMap map[string]*Struct //key是StructName
	FunctionManager
	// file      *GenedFile
}

type Import struct {
	Name string
	Path string
}

func CreatePackage(project *Project, modPath string) *Package {
	return &Package{
		Project:         project,
		StructMap:       make(map[string]*Struct),
		FunctionManager: createFunctionManager(),
		PackageInfo: PackageInfo{
			modPath: modPath,
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
		fmt.Printf("begin parse %s with %s\n", packName, path)
		for filename, f := range pack.Files {
			pkg.parseMod(f, filename)
			gofile := createGoFile(f, pkg, filename)
			gofile.parseFile()
		}
	}
}

func (pkg *Package) parseMod(file *ast.File, fileName string) {
	if len(pkg.modName) == 0 {
		pkg.modName = file.Name.Name
	} else {
		// 多个文件的mod应该相同，否则报错
		if pkg.modName != file.Name.Name {
			log.Fatalf("mod of %s is %s which should be %s\n", fileName, file.Name.Name, pkg.modName)
		}
	}
}

func (pkg *Package) getStruct(name string, create bool) *Struct {
	var class *Struct
	var ok bool
	if class, ok = pkg.StructMap[name]; !ok {
		if create {
			class = CreateStruct(name, pkg)
			pkg.StructMap[name] = class
		}
	}
	return class
}

// 生成代码
func (pkg *Package) generateInitorCode(file *GenedFile) (define, assign strings.Builder) {
	for _, initors := range pkg.initiatorMap {
		for _, variable := range initors.list {
			define.WriteString(variable.genDefinition(file))
			define.WriteString("\n")
			name := variable.name
			assign.WriteString(name)
			assign.WriteString("=")
			assign.WriteString(variable.generateCode("", file))
			assign.WriteString("\n")
		}
	}
	return
}
func (pkg *Package) GenerateCode() (initorName, routerName string) {
	// 产生文件；
	file := createGenedFile()

	var routerFunction strings.Builder
	// 调用initiator函数
	define, assign := pkg.generateInitorCode(&file)
	// 针对每个struct，产生servlet文件；
	for _, class := range pkg.StructMap {
		if len(class.servletMethods) > 0 {
			routerFunction.WriteString(class.GenerateCode(&file))
		}
	}
	if define.Len() == 0 && routerFunction.Len() == 0 {
		return
	}
	var name string
	if len(pkg.modPath) > len(pkg.Project.Mod) {
		name = pkg.modPath[len(pkg.Project.Mod)+1:]
	} else {
		name = pkg.modName
	}
	name = strings.ReplaceAll(name, "/", "_")
	var content strings.Builder
	content.WriteString("package gen\n")
	content.WriteString(file.genImport())
	if define.Len() > 0 {
		initorName = fmt.Sprintf("init%s_variable", name)
		content.WriteString(define.String())
		content.WriteString("func " + initorName + "(){\n")
		content.WriteString(assign.String())
		content.WriteString("}\n")
	}
	if routerFunction.Len() > 0 {
		routerName = fmt.Sprintf("init%s_router", name)
		content.WriteString("func " + routerName + "(router *gin.Engine){\n")
		content.WriteString(routerFunction.String())
		content.WriteString("}\n")
	}

	os.WriteFile(name+".go", []byte(content.String()), 0660)
	return
}
