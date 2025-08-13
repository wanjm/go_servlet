package astinfo

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strings"
)

type Gosourse struct {
	Path string
	Pkg  *Package
	File *ast.File
	// import a "github.com/abc"
	// key = a ; value = github.com/abc
	Imports map[string]string //key package name,value是包路径；
	// key=github.com/abc; value=a
	NoNameImport map[string]string //key 是module，value是包名；
}

// parseType
func (g *Gosourse) parseType(typeSpec *ast.TypeSpec) {
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType:
		class := NewInterface(g, typeSpec)
		g.Pkg.AddParser(class)
	case *ast.StructType:
		// fmt.Printf("StructType %s %s\n", typeSpec.Name.Name, g.Path)
		class := NewStruct(g, typeSpec)
		g.Pkg.AddParser(class)
	default:
		alias := NewAlias(typeSpec, g, typeSpec.Assign != 0)
		g.Pkg.AddParser(alias)
	}
}

// FindPackage 查找package
func (g *Gosourse) FindPackage(name string) *Package {
	module := g.Imports[name]
	if module == "" {
		for _, module := range g.NoNameImport {
			pkg := GlobalProject.FindPackage(module)
			g.Imports[pkg.Name] = module
		}
	}
	return GlobalProject.FindPackage(module)
}

// 解析全局变量和 struct，interface
func (g *Gosourse) getGenDeclParser(genDecl *ast.GenDecl) (parser Parser) {
	switch genDecl.Tok {
	case token.VAR:
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.ValueSpec)
			parser = NewVarFieldHelper(typeSpec, g)
			g.Pkg.AddParser(parser)
		}
	case token.TYPE:
		//如果是一个Specs，那么是一个type定义模式；则将其送到Specs[0].Doc中；
		if len(genDecl.Specs) == 1 {
			typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
			typeSpec.Doc = genDecl.Doc
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			g.parseType(typeSpec)
		}
	}
	return
}

// 在一个go文件中解析一个类型，其可能为:
// 原始类型； string
// 结构体的范型参数；
// 同package的结构体，
// field.Type =
// 先检查原始类型；
// 来自 import . "****"
func (g *Gosourse) getType(typeName string, typeMap map[string]*Field) Typer {
	//check raw type
	type1 := GetRawType(typeName)
	if type1 != nil {
		return type1
	}
	//check typeMap
	type2 := typeMap[typeName]
	if type2 != nil {
		return type2.Type
	}
	//check pkg
	type3 := g.Pkg.GetTyper(typeName)
	if type3 != nil {
		return type3
	}
	// check import .
	pkg := g.FindPackage(".")
	if pkg != nil {
		type3 = pkg.GetTyper(typeName)
		if type3 != nil {
			return type3
		}
	}
	fmt.Printf("failed to find type %s in %s\n", typeName, g.Path)
	return nil
}

// 解析函数和方法
func (g *Gosourse) getFuncDeclParser(funcDecl *ast.FuncDecl) Parser {
	switch funcDecl.Recv {
	case nil:
		return NewFunctionParserHelper(funcDecl, g)
	default:
		return NewMethod(funcDecl, g)
	}
}

//	func (g *Gosourse) ParseTop() bool {
//		for _, c := range g.File.Comments {
//			for _, line := range c.List {
//				if strings.Contains(line.Text, "//go:build ignore") ||
//					strings.Contains(line.Text, "// +build ignore") {
//					return false
//				}
//			}
//		}
//		// TODO: 需要添加日志级别，再打印日志
//		// fmt.Printf("Parsing file: %s name: %s %s\n", g.Path, g.Pkg.Name, g.Pkg.Module)
//		g.parseImport(g.File.Imports)
//		decls := g.File.Decls
//		for i := 0; i < len(decls); i++ {
//			switch decl := decls[i].(type) {
//			case *ast.GenDecl:
//				g.getGenDeclParser(decl)
//			}
//		}
//		return true
//	}
func isIgnoreFile(file *ast.File) bool {
	for _, c := range file.Comments {
		for _, line := range c.List {
			if strings.Contains(line.Text, "//go:build ignore") ||
				strings.Contains(line.Text, "// +build ignore") {
				return true
			}
		}
	}
	return false
}
func (g *Gosourse) Parse() error {
	g.parseImport(g.File.Imports)
	decls := g.File.Decls
	for i := 0; i < len(decls); i++ {
		switch decl := decls[i].(type) {
		case *ast.GenDecl:
			g.getGenDeclParser(decl)
		case *ast.FuncDecl:
			p := g.getFuncDeclParser(decl)
			if p != nil {
				g.Pkg.AddParser(p)
			}
		}
	}
	// 方法体为空
	return nil
}

func NewGosourse(file *ast.File, pkg *Package, path string) *Gosourse {
	return &Gosourse{
		File:         file,
		Pkg:          pkg,
		Path:         path,
		Imports:      map[string]string{},
		NoNameImport: map[string]string{},
	}
}

// 解析go文件的Import字段，如果有modeName直接使用，否则用pathValue的文件名；
// 注意此处可能有错误，因为有些package的模块名不是路径的最后一位；
// 此时只能通过解析原package文件才能解决；否则后面getImportPath就找不到了
func (goFile *Gosourse) parseImport(imports []*ast.ImportSpec) {
	for _, importSpec := range imports {
		var name string
		modulePath := importSpec.Path.Value
		pathValue := strings.Trim(modulePath, string(modulePath[0]))
		if pathValue == "C" {
			continue
		}
		if importSpec.Name != nil {
			name = importSpec.Name.Name
		} else {
			name = path.Base(pathValue)
			// 临时用base代替，仅有小部分不是baseName，等到查不到的时候再查找；
			// 只有name不存在时，才临时使用；避免错误覆盖；
			goFile.NoNameImport[pathValue] = name
			if _, ok := goFile.Imports[name]; ok {
				continue
			}
			// 如果没有带名字，则从Package中寻找，此处是否可能该Package还没有被解析呢？
			// pkg := GlobalProject.FindPackage(pathValue)
			// name = pkg.GetName()
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
