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
	// Struct    []*Struct
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
		fmt.Printf("begin parse %s with %s\n", packName, path)
		for filename, f := range pack.Files {
			pkg.parseMod(f, filename)
			gofile := createGoFile(f, pkg, filename)
			gofile.parseFile()
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
	var sb strings.Builder
	for _, class := range pkg.StructMap {
		if len(class.ServletMethods) > 0 {
			sb.WriteString(class.GenerateCode())
		}
	}
	return sb.String()
}
