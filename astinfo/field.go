package astinfo

import (
	"go/ast"
	"strings"
)

const (
	TypeInterface = iota
	TypeStruct
)

// 定义了Struct中的一个个属性, 也用于函数的参数和返回值
type Field struct {
	class     interface{}
	typeName  string   //类型名
	pkg       *Package //class所在的包
	isPointer bool
	name      string
	jsonName  string
	comment   string
	ownerInfo string //记录用于打印日志的信息
}

func (field *Field) parse(astField *ast.Field, goFile *GoFile) {
	field.parseTag(astField.Tag, goFile)
	field.parseComment(astField.Comment, goFile)
	field.parseType(astField.Type, goFile)
}
func (field *Field) parseTag(fieldType *ast.BasicLit, goFile *GoFile) {
	if fieldType != nil {
		tag := strings.Trim(fieldType.Value, "`\"")
		tagList := strings.Split(tag, " ")
		for _, tag := range tagList {
			if strings.Contains(tag, "json") {
				value := strings.Trim(strings.Split(tag, ":")[1], "\"")
				field.jsonName = strings.Split(value, ",")[0]
			}
		}
	}
}
func (field *Field) parseComment(fieldType *ast.CommentGroup, goFile *GoFile) {
	if fieldType != nil && len(fieldType.List) > 0 {
		field.comment = strings.Trim(fieldType.List[0].Text, "\" /")
	}
}
func (field *Field) parseType(fieldType ast.Expr, goFile *GoFile) {
	var modeName, structName string
	// 内置slice类型；
	if arrayType, ok := fieldType.(*ast.ArrayType); ok {
		field.typeName = "array"
		field.pkg = goFile.pkg.Project.rawPkg

		fakeFiled := Field{}
		fakeFiled.parseType(arrayType.Elt, goFile)
		array := ArrayType{}
		if fakeFiled.class != nil {
			array.OriginType = fakeFiled.class.(SchemaType)
		} else {
			array.pkg = fakeFiled.pkg
			array.typeName = fakeFiled.typeName
		}
		field.class = &array
		return
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
	// 原生类型，或者本package定义的结构体,array在前面已经处理了，所以此处肯定没有数组；
	// 下面的class也可以直接使用
	if innerType, ok := fieldType.(*ast.Ident); ok {
		structName = innerType.Name
		if structName[0] <= 'z' && structName[0] >= 'a' {
			project := goFile.pkg.Project
			class := project.getStruct(structName, nil, nil)
			if class != nil {
				field.typeName = structName
				field.pkg = project.rawPkg
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

	// 由于一个变量定义类型可能是结构体，也可能是接口，所以此处不能直接获取结构体
	// 而且此处不尝试获取接口或者结构，是为了避免由于代码写法不同，导致的可能找不到，可能找到的情况；简化了代码场景；
	field.pkg = goFile.pkg.Project.getPackage(pkgPath, true)
	field.typeName = structName
	// field.class = pkg.getStruct(structName, true)
}
func (field *Field) generateCode() string {
	return "\n"
}

// 再给vairable赋值时，强行force为true；
// 为什么有些是结构体，不过不强行却找不到：如外部结构体，由于本代码不会扫描到外部结构体，所以找不到；
func (field *Field) findStruct(force bool) *Struct {
	// 此处如果代码错误，会出现class为Interface，但是强转为Struct的情况，让程序报错
	if field.class == nil {
		field.class = field.pkg.getStruct(field.typeName, force)
	}
	if a, ok := field.class.(*Struct); ok {
		return a
	}
	return nil
}

func (field *Field) findInterface() *Interface {
	if field.class == nil {
		field.class = field.pkg.getInterface(field.typeName, false)
	}
	if a, ok := field.class.(*Interface); ok {
		return a
	}
	return nil
}
