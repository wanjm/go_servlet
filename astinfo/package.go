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
	// file      *GenedFile
}

type Import struct {
	Name string
	Path string
}

func CreatePackage(project *Project, modPath string) *Package {
	return &Package{
		Project:   project,
		StructMap: make(map[string]*Struct),
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

func (pkg *Package) GenerateCode() string {
	file := createGenedFile()
	var sb strings.Builder
	for _, class := range pkg.StructMap {
		if len(class.ServletMethods) > 0 {
			sb.WriteString(class.GenerateCode(&file))
		}
	}
	if sb.Len() == 0 {
		return ""
	}
	name := pkg.modPath[len(pkg.Project.Mod)+1:]
	name = strings.ReplaceAll(name, "/", "_")
	content := ("package gen\n" +
		file.genImport() +
		"func init" + name + "(router *gin.Engine){\n") +
		sb.String() +
		"}\n"
	os.WriteFile(name+".go", []byte(content), 0660)
	return fmt.Sprintf("init%s(router)\n", name)
}
