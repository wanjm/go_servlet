package astinfo

import (
	"go/ast"
	"strings"
)

const (
	TypeInterface = iota
	TypeStruct
)

type FieldComment struct {
	defaultValue string //记录该属性的默认值，在struct的field中有使用；
	// isRequired   bool   //记录该字段是否必须赋值，区别于gin的默认处理方法，必传表示在报文中必须存在
	validString string //校验变量是否符合要求的代码； $>10 && $<11
	comment     string
}

func (comment *FieldComment) dealValuePair(key, value string) {
	switch key {
	case "default":
		comment.defaultValue = value
	case "valid":
		comment.validString = value
	default:
		comment.comment = key
	}
}

// 定义了Struct中的一个个属性, 也用于函数的参数和返回值
type Field struct {
	class     interface{}
	typeName  string   //类型名
	pkg       *Package //class所在的包
	isPointer bool
	name      string
	jsonName  string
	ownerInfo string //记录用于打印日志的信息
	creators  map[*Struct]*Function
	comment   FieldComment
}

func (field *Field) parse(astField *ast.Field, goFile *GoFile) {
	field.parseTag(astField.Tag, goFile)
	field.parseComment(astField.Comment, goFile)
	field.parseType(astField.Type, goFile)
}
func (field *Field) parseTag(fieldType *ast.BasicLit, _ *GoFile) {
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
func (field *Field) parseComment(fieldType *ast.CommentGroup, _ *GoFile) {
	if fieldType == nil || len(fieldType.List) <= 0 {
		return
	}
	content := strings.Trim(fieldType.List[0].Text, " /")
	// field.comment =
	parseValidComment(content, &field.comment)
}
func (field *Field) parseType(fieldType ast.Expr, goFile *GoFile) {
	var modeName, structName string
	// 内置slice类型；
	if arrayType, ok := fieldType.(*ast.ArrayType); ok {
		// 内置array类型
		// field的pkg指向原始类型；
		// field的class只想ArrayType;
		// ArrayType中的pkg，typeName，class指向具体的类型
		field.typeName = "array"
		field.pkg = goFile.pkg.Project.rawPkg

		fakeFiled := Field{}
		fakeFiled.parseType(arrayType.Elt, goFile)
		array := ArrayType{}
		if fakeFiled.class != nil {
			array.class = fakeFiled.class.(SchemaType)
		}
		array.pkg = fakeFiled.pkg
		array.typeName = fakeFiled.typeName
		array.isPointer = fakeFiled.isPointer
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

// 产生给本变量赋值的方法， 方式等号的右边；
// 给结构体赋值，则是在冒号的后面
// field= *hello;  field=hello , field=getAddr(hello);
// 当field是普通类型，且没有默认值时，返回""; 当前应该只有构建结构体时才遇到。如果字符串的值是“”，此处返回"\"\""
func (field *Field) generateCode(receiverPrefix string, file *GenedFile) string {
	var res string
	var isVariablePointer = false
	if field.pkg == field.pkg.Project.rawPkg {
		// rawPackage 由于是原始值，所以variable都不是pointer
		res = field.comment.defaultValue
		if res == "" {
			return res
		}
	} else {
		variable := Variable{
			isPointer: field.isPointer,
			class:     field.findStruct(true),
			name:      field.name,
		}
		// 从receiver中查找是否有Creator方法
		creator := field.creators[variable.class]
		if creator != nil {
			variable.creator = creator
			variable.isPointer = creator.Results[0].isPointer
		}
		res = variable.generateCode(receiverPrefix, file)
		isVariablePointer = variable.isPointer
	}
	if field.isPointer == isVariablePointer {
		return res
	} else if isVariablePointer {
		return "*" + res
	} else {
		return "getAddr(" + res + ")"
	}
}

// 再给vairable赋值时，强行force为true；
// 为什么有些是结构体，不过不强行却找不到：如外部结构体，由于本代码不会扫描到外部结构体，所以找不到；
// 参考：parseType中的注释
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

// prefix 为前缀，用于生成代码时，是自己带.的
func (field *Field) genCheckArrayNil(prefix string, file *GenedFile, content *strings.Builder) {
	if field.typeName == "array" {
		arrayType := field.class.(*ArrayType)
		content.WriteString("if " + prefix + field.name + " == nil {\n")
		content.WriteString(prefix + field.name + "= []")
		if arrayType.isPointer {
			content.WriteString("*")
		}
		if arrayType.pkg == field.pkg.Project.rawPkg {
			// 原始类型
			content.WriteString(arrayType.typeName)
		} else {
			class := arrayType.findStruct(true)
			content.WriteString(class.generate0Slice(file))

		}
		content.WriteString("{}\n}\n")
	} else if field.pkg == field.pkg.Project.rawPkg {

	} else {
		class := field.findStruct(true)
		if field.isPointer {
			// 支持指针类型，但是仅支持原始类型，type alias后暂时不支持；
			content.WriteString("if " + prefix + field.name + "!= nil {\n")
		}
		class.genCheckNil(prefix+field.name+".", file, content)
		if field.isPointer {
			content.WriteString("}\n")
		}
	}
}
