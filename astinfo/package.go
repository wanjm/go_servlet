package astinfo

import (
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
// 初始化函数可以自带参数，这些参数是由其他initiator生成的，所以initiator之间是有依赖关系的，需要生成代码时，进行排序，保证生成顺序的正确性；
func (pkg *Package) generateInitorCode(manager *InitiatorManager, file *GenedFile) {
	if len(pkg.initiators) == 0 {
		return
	}
	define := strings.Builder{}
	assign := strings.Builder{}
	file.addBuilder(&define)
	file.addBuilder(&assign)
	var level = 0
	for _, node := range pkg.initiators {
		if node.level > level {
			level = node.level
		}
		initor := node.function
		variable := node.returnVariable
		if variable != nil {
			define.WriteString(variable.genDefinition(file))
			define.WriteString("\n")
			assign.WriteString(variable.name)
			assign.WriteString("=")
		}
		assign.WriteString(initor.genCallCode("", pkg.file))
		assign.WriteString("\n")
	}
	assign.WriteString("}\n")
}

func (pkg *Package) getInitiatorFunctionName() string {
	return "init_" + pkg.file.name + "_variable"
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
	// init 函数有多个，每个servlet_group一个函数；
	var builders = make(map[string]*strings.Builder)
	// maps.Keys[]()
	keys := getSortedKey(pkg.StructMap)
	for _, key := range keys {
		class := pkg.StructMap[key]
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
