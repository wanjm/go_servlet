package astinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
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
	Project      *Project
	StructMap    map[string]*Struct    //key是StructName
	InterfaceMap map[string]*Interface //key是Interface 的Name
	FunctionManager
	file *GenedFile
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
		InterfaceMap:    make(map[string]*Interface),
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
		_ = packName
		// fmt.Printf("begin parse %s with %s\n", packName, path)
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

func (pkg *Package) getInterface(name string, create bool) *Interface {
	var class *Interface
	var ok bool
	if class, ok = pkg.InterfaceMap[name]; !ok {
		if create {
			class = CreateInterface(name, pkg)
			pkg.InterfaceMap[name] = class
		}
	}
	return class
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

// 生成initiator的代码
// 定义initorator的函数，在此被调用，并保存在全局变量中；
func (pkg *Package) generateInitorCode() {
	if len(pkg.initiators) == 0 {
		return
	}
	define := strings.Builder{}
	assign := strings.Builder{}
	pkg.file.addBuilder(&define)
	pkg.file.addBuilder(&assign)
	initorName := fmt.Sprintf("init%s_variable", pkg.file.name)
	assign.WriteString("func " + initorName + "(){\n")
	pkg.Project.addInitVariable(initorName)
	for _, initor := range pkg.initiators {
		if len(initor.Results) == 1 {
			result := initor.Results[0]
			name := result.name
			if len(name) == 0 {
				name = strings.ReplaceAll(result.pkg.modPath, ".", "_")
				name = strings.ReplaceAll(name, "/", "_")
			}

			variable := Variable{
				// creator:   initor,
				class:     result.findStruct(true),
				name:      name,
				isPointer: result.isPointer,
			}
			//先添加到全局定义中去，可能给variable补名字
			pkg.Project.addInitiatorVaiable(&variable)
			define.WriteString(variable.genDefinition(pkg.file))
			define.WriteString("\n")

			assign.WriteString(name)
			assign.WriteString("=")
		}
		assign.WriteString(initor.genCallCode("", pkg.file))
		assign.WriteString("\n")
	}
	assign.WriteString("}\n")
}

// 生成rpc客户端的代码
func (pkg *Package) GenerateRpcClientCode() {
	var rpcBuilder strings.Builder
	var hasContent = false
	for _, rpc := range pkg.InterfaceMap {
		// interface的方法保存在servlets中
		if rpc.GenerateCode(pkg.file, &rpcBuilder) {
			hasContent = true
		}
	}
	if hasContent {
		pkg.file.addBuilder(&rpcBuilder)
	}
}
func (pkg *Package) GenerateRouteCode() {
	// 产生文件；
	// 针对每个struct，产生servlet文件；
	var builders = make(map[string]*strings.Builder)
	for _, class := range pkg.StructMap {
		if len(class.servlets) > 0 {
			routerName := "init_" + class.comment.groupName + "_" + pkg.file.name + "_router"
			var builder *strings.Builder
			var ok bool
			if builder, ok = builders[routerName]; !ok {
				builder = &strings.Builder{}
				builders[routerName] = builder
				builder.WriteString("func " + routerName + "(router *gin.Engine){\n")
				pkg.file.addBuilder(builder)
				pkg.Project.addInitRoute(routerName, class.comment.groupName)
			}
			builder.WriteString(class.GenerateCode(pkg.file))
		}
	}
	for _, builder := range builders {
		builder.WriteString("}\n")
	}
}
