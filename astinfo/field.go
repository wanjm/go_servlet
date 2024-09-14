package astinfo

import (
	"go/ast"
)

// 定义了Struct中的一个个属性
type Field struct {
	class     *Struct
	isPointer bool
	name      string
	ownerInfo string
}

func (field *Field) parse(fieldType ast.Expr, goFile *GoFile) {
	var modeName, structName string
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
			class := goFile.pkg.Project.getPackage(GolangRawType, false).getStruct(structName, false)
			if class != nil {
				field.class = class
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
	pkg := goFile.pkg.Project.getPackage(pkgPath, true)
	field.class = pkg.getStruct(structName, true)
}
func (field *Field) generateCode() string {

	return "\n"
}
