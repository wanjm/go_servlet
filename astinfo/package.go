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
	define strings.Builder
	assign strings.Builder
	file   *GenedFile
}

type Import struct {
	Name string
	Path string
}

// pkgInfo的ModName没有填写，等到解析package再填写，这样便于发现源代码的bug；
// 理论上源代码有bug也不需要我们来发现；但是先这么做；
// 所以对于第三方的Packge，是没有机会生成modeName的；
// 在最终生成代码时，如果为空，则用modePaht的basename来替换。不影响import语句的生成；
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
func (pkg *Package) generateInitorCode() {
	define := &pkg.define
	assign := &pkg.assign
	for _, initor := range pkg.initiators {
		result := initor.Results[0]
		variable := Variable{
			creator:   initor,
			class:     result.class,
			name:      result.name,
			isPointer: result.isPointer,
		}
		//先添加到全局定义中去，可能给variable补名字
		pkg.Project.addInitiatorVaiable(&variable)
		define.WriteString(variable.genDefinition(pkg.file))
		define.WriteString("\n")
		name := variable.name
		assign.WriteString(name)
		assign.WriteString("=")
		assign.WriteString(variable.generateCode("", pkg.file))
		assign.WriteString("\n")
	}
}
func (pkg *Package) GenerateCode() (initorName, routerName string) {
	// 产生文件；
	var routerFunction strings.Builder
	// 调用initiator函数
	// 针对每个struct，产生servlet文件；
	for _, class := range pkg.StructMap {
		if len(class.servlets) > 0 {
			routerFunction.WriteString(class.GenerateCode(pkg.file))
		}
	}
	define := &pkg.define
	assign := &pkg.assign
	if define.Len() == 0 && routerFunction.Len() == 0 {
		return
	}
	var name string
	name = pkg.Project.getRelativeModePath(pkg.modPath)
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	var content strings.Builder
	content.WriteString("package gen\n")
	content.WriteString(pkg.file.genImport())
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
