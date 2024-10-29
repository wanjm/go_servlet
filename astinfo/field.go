package astinfo

import (
	"go/ast"
)

const (
	TypeInterface = iota
	TypeStruct
)

// 定义了Struct中的一个个属性
type Field struct {
	class     interface{}
	typeName  string
	pkg       *Package
	isPointer bool
	name      string
	ownerInfo string
}

func (field *Field) parse(fieldType ast.Expr, goFile *GoFile) {
	var modeName, structName string
	// 内置slice类型；
	if _, ok := fieldType.(*ast.ArrayType); ok {
		rawPkg := goFile.pkg.Project.getPackage(GolangRawType, false)
		class := rawPkg.getStruct("array", false)
		if class != nil {
			field.typeName = "array"
			field.pkg = rawPkg
			return
		}
	}
	if innerType, ok := fieldType.(*ast.StarExpr); ok {
		field.isPointer = true
		fieldType = innerType.X
	}
	var pkgPath string
	if innerType, ok := fieldType.(*ast.SelectorExpr); ok {
		modeName = innerType.X.(*ast.Ident).Name
		structName = innerType.Sel.Name
		pkgPath = goFile.getImportPath(modeName, field.ownerInfo)
	}
	// 原生类型，或者本package定义的结构体
	if innerType, ok := fieldType.(*ast.Ident); ok {
		structName = innerType.Name
		if structName[0] <= 'z' && structName[0] >= 'a' {
			rawPkg := goFile.pkg.Project.getPackage(GolangRawType, false)
			class := rawPkg.getStruct(structName, false)
			if class != nil {
				field.typeName = structName
				field.pkg = rawPkg
				return
			}
		}
		pkgPath = goFile.pkg.modPath
	}
	// 此处有三种情况
	// 1. 返回一个本项目存在结构体，mymode.Struct
	// 2. 返回一个第三方的结构体体
	// 3. 返回一个本pkg的结构体，Struct
	// 4. 原生类型，int，string
	field.pkg = goFile.pkg.Project.getPackage(pkgPath, true)
	field.typeName = structName
	// field.class = pkg.getStruct(structName, true)
}
func (field *Field) generateCode() string {
	return "\n"
}

func (field *Field) findStruct() *Struct {
	// 此处如果代码错误，会出现class为Interface，但是强转为Struct的情况，让程序报错
	if field.class == nil {
		field.class = field.pkg.getStruct(field.typeName, false)
	}
	return field.class.(*Struct)
}

func (field *Field) findInterface() *RpcInterface {
	if field.class == nil {
		field.class = field.pkg.getRpcInterface(field.typeName, false)
	}
	return field.class.(*RpcInterface)
}
