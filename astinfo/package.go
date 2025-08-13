package astinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strings"
)

// Package 表示一个Go包的基本信息
type Package struct {
	Simple bool   // 简单解析，及仅解析包名
	Name   string // 包名称
	Path   string // 包所在目录的绝对路径
	Module string // 所属模块全路径
	// 用于变量注入的检查，用于servlet的生成；
	Structs map[string]*Struct // 包内结构体集合（key为结构体名称）
	// Interfaces map[string]*Interface // key是Interface 的Name
	// 由于采用了两层扫描，所以不再需要Types map了。直接调用get方法获取；
	parsers   []Parser             // 先扫描文件，生成parsers,然后依次进行parser解析；
	Types     map[string]Typer     // key是Type 的Name
	fset      *token.FileSet       // 记录fset，到时可以找到文件
	Files     map[string]*ast.File // Go source files by filename
	GlobalVar map[string]*VarField
	WaitTyper map[string][]*Typer // 有些类型先被使用，再定义，此时在此处将内容缓存下载，最后统一解析；
	FunctionManager
	finshedParse bool
}

//  Package中不包含goSource，因为
// 1. package 仅仅是因为需要被引用时生成；
// 2. package 有多个goSource文件；
// 3. package 将来可以建立goSource和ast.File之间的映射关系来达到寻找关系的目的；

// getName
func (pkg *Package) GetName() string {
	name := pkg.Name
	// Module Name有可能跟包名不一样，
	// 本模块需要解析的包都是有name的。但是第三方的包可能就没有，需要从Module Name中解析
	// 等后续有需求了再做；
	if name == "" {
		fmt.Printf("failed to get name of pkg %s, use base name in path which is error\n", pkg.Module)
		name = filepath.Base(pkg.Module)
	}
	return name
}

// addParser
func (pkg *Package) AddParser(parser Parser) {
	pkg.parsers = append(pkg.parsers, parser)
}

// 采用遇到遇到不认识的import就先深度parse的方法；
func (pkg *Package) SimpleParse() error {

	path := pkg.Path
	defer func() {
		for name, alias := range pkg.WaitTyper {
			typer := pkg.GetTyper(name)
			if typer != nil {
				for _, typer1 := range alias {
					*typer1 = typer
				}
			} else {
				fmt.Printf("Error: failed to get %s.%s when parse finish\n", pkg.Path, name)
			}
		}
		// fmt.Printf("finished Parsing package: %s\n", path)
	}()
	pkg.fset = token.NewFileSet()
	// 这里取绝对路径，方便打印出来的语法树可以转跳到编辑器
	// fmt.Printf("Parsing package: %s\n", path)
	packageMap, err := parser.ParseDir(pkg.fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Printf("parse package %s failed %s\n", pkg.Module, err.Error())
		return nil
	}
	// 一个目录下可能有多
	for packName, pack := range packageMap {
		_ = packName
		// 先跳过test,后续再说；
		if strings.HasSuffix(packName, "_test") {
			continue
		}
		for _, f := range pack.Files {
			if isIgnoreFile(f) {
				continue
			}
			pkg.Name = pack.Name
			pkg.Files = pack.Files
			break
		}
	}
	return nil
}
func (pkg *Package) Parse() error {
	if pkg.finshedParse {
		return nil
	}
	fmt.Printf("parse package %s\n", pkg.Module)
	pkg.finshedParse = true
	for filename, f := range pkg.Files {
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		gofile := NewGosourse(f, pkg, filename)
		gofile.Parse()
	}
	if pkg.Simple {
		for _, parser := range pkg.Types {
			parser.Parse()
		}
	} else {
		for _, parser := range pkg.parsers {
			parser.Parse()
		}
	}

	return nil
}

// NewPackage creates a new Package instance with the given module path
func NewPackage(module string, simple bool, absPath string) *Package {
	// Extract package name from module path
	return &Package{
		Module:  module,
		Simple:  simple,
		Path:    absPath,
		Structs: make(map[string]*Struct),
		// Interfaces: make(map[string]*Interface),
		GlobalVar: make(map[string]*VarField),
		Types:     make(map[string]Typer),
		WaitTyper: make(map[string][]*Typer),
		// finshedParse: simple,
	}
}

func NewSysPackage(module string) *Package {
	return NewPackage(module, true, path.Join("/opt/google/go/src", module))
}

func (pkg *Package) GetTyper(name string) Typer {
	if !pkg.finshedParse {
		pkg.Parse()
	}
	return pkg.Types[name]
}

func SimplePackage(module, name string) *Package {
	return &Package{
		Module: module,
		Name:   name,
	}
}
