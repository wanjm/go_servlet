package astinfo

import (
	"go/ast"
	"strings"

	"github.com/go-openapi/spec"
)

// /@goservlet prpc=xxx; servlet=xxx; servle; prpc
type structComment struct {
	groupName  string
	serverType int // NONE, RpcStruct, ServletStruct
}

func (comment *structComment) dealValuePair(key, value string) {
	switch key {
	case Prpc:
		comment.serverType = PRPC
		if len(value) == 0 {
			comment.groupName = Prpc
		} else {
			comment.groupName = value
		}
	case Servlet:
		comment.serverType = SERVLET
		if len(value) == 0 {
			comment.groupName = Servlet
		} else {
			comment.groupName = value
		}
	}
}

type Struct struct {
	Name string
	// ImportUrl      string
	// ServletMethods []*Method           //记录路由代码
	// CreatorMethods map[*Struct]*Method //纪录构建默认参数的代码, key是构建的struct
	FunctionManager
	// Import         *Import
	Package     *Package
	structFound bool
	fields      []*Field
	usage       int
	comment     structComment
	// 自动生成代码相关参数，此处可能需要更改为StructObject对象
	ref *spec.Ref
}

func CreateStruct(name string, pkg *Package) *Struct {
	return &Struct{
		Name:            name,
		Package:         pkg,
		FunctionManager: createFunctionManager(),
	}
}
func (class *Struct) parse(structType *ast.StructType, goFile *GoFile) {
	for _, field := range structType.Fields.List {
		classField := Field{
			ownerInfo: "struct Name is " + class.Name,
		}
		classField.parse(field, goFile)
		for _, name := range field.Names {
			oneClassField := classField
			oneClassField.name = name.Name
			class.fields = append(class.fields, &oneClassField)
		}
	}
}

func (class *Struct) parseComment(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	return ""
}

// 注意跟变量注入区分开来
func (class *Struct) GenerateCode(file *GenedFile) string {

	if len(class.servlets) == 0 {
		return ""
	}
	receiver := &Variable{
		class:     class,
		isPointer: false,
		name:      firstLower(class.Name),
	}
	var sb strings.Builder
	// 定义strutct对象；
	sb.WriteString(receiver.name + ":=" + receiver.generateCode("", file) + "\n")
	for _, servlet := range class.servlets {
		switch servlet.comment.funcType {
		case SERVLET:
			sb.WriteString(servlet.GenerateServlet(file, receiver.name+"."))
		case PRPC:
			sb.WriteString(servlet.GenerateRpcServlet(file, receiver.name+"."))
		case WEBSOCKET:
			sb.WriteString(servlet.GenerateWebsocket(file, receiver.name+"."))
		}
	}
	return sb.String()
}

// 该方法会用于生成变量的代码；
// 1. 生成用于servet的类的对象；
// 2. 用于生成servlet参数的对象；
// 是否生成注入的代码，需要考虑 上述1，2的注入方法是否有区别
// func (class *Struct) generateObject() string {
// 	// 变量名的规则是 ${modName}${struct.Name}
// 	codeFmt := "%s.%s{}\n"
// 	return fmt.Sprintf(codeFmt, class.Package.ModInfo.Name, class.Name)
// }

// func (class *Struct) addServlet(method *Method) {
// 	class.ServletMethods = append(class.ServletMethods, method)
// }

// // creator是用户提供的创建某个对象的方法，主要是用于设置请求的默认值；主要用于构建servlet的request参数
// func (class *Struct) addCreator(childClass *Struct, method *Method) {
// 	class.CreatorMethods[childClass] = method
// }

// 一个servlet的request对象，可以直接构造空方法，也可以调用该类型提供的creator方法；
// func (class *Struct) GetCreatorCode4Struct(childClass *Struct) string {
// 	if method, ok := class.CreatorMethods[childClass]; ok {
// 		// 调用cratetor方法，则为该对象的变量+creator方法
// 		return class.variableName + "." + method.Name + "()\n"
// 	} else {
// 		// 直接构建空对象
// 		return childClass.generateObject()
// 	}
// }
