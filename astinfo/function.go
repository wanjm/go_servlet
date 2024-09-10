package astinfo

import (
	"fmt"
	"go/ast"
	"log"
	"regexp"
	"strings"
)

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
	INITIATOR
)

type FunctionManag interface {
	addServlet(*Function)
	addCreator(childClass *Struct, method *Function)
	addInitiator(initiator *Function)
}

type FunctionManager struct {
	creators   map[*Struct]*Function //纪录构建默认参数的代码, key是构建的struct
	initiators []*Function           //初始化函数
	servlets   []*Function           //记录路由代码

}

func createFunctionManager() FunctionManager {
	return FunctionManager{
		creators: make(map[*Struct]*Function),
	}
}

func (funcManager *FunctionManager) addServlet(function *Function) {
	funcManager.servlets = append(funcManager.servlets, function)
}

func (funcManager *FunctionManager) addCreator(childClass *Struct, function *Function) {
	funcManager.creators[childClass] = function
}

// 入参直接是函数返回值的对象，跟method.Result[0]相同,为了保持返回值的variable不受影响
func (funcManager *FunctionManager) addInitiator(initiator *Function) {
	funcManager.initiators = append(funcManager.initiators, initiator)
}

func (funcManager *FunctionManager) getCreator(childClass *Struct) (function *Function) {
	return funcManager.creators[childClass]
}

type Function struct {
	Name        string      // method name
	Params      []*Variable // method params, 下标0是request
	Results     []*Variable // method results（output)
	function    *ast.FuncDecl
	pkg         *Package
	goFile      *GoFile
	funcManager *FunctionManager

	Url        string // method url from comments;
	deprecated bool
}

func createFunction(f *ast.FuncDecl, goFile *GoFile) *Function {
	return &Function{
		function:    f,
		pkg:         goFile.pkg,
		goFile:      goFile,
		funcManager: &goFile.pkg.FunctionManager,
	}
}

// 解析注释
func (function *Function) parseComment() int {
	f := function.function
	function.Name = f.Name.Name
	funcType := NOUSAGE
	// isCreator := strings.HasSuffix(method.Name, "Creator")
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			text := strings.Trim(comment.Text, "/ \t") // 去掉前后的空格和斜杠
			text = strings.ReplaceAll(text, "\t ", "")
			if strings.HasPrefix(text, TagPrefix) {
				pattern := regexp.MustCompile(`\s+=\s+`)
				newString := pattern.ReplaceAllString(text[len(TagPrefix):], "=")
				commands := strings.Split(newString, " ")
				for _, command := range commands {
					valuePair := strings.Split(command, "=")
					if len(valuePair) == 2 {
						valuePair[1] = strings.Trim(valuePair[1], "\"'")
					}
					switch valuePair[0] {
					case "url":
						function.Url = valuePair[1]
						return SERVLET
					case "creator":
						return CREATOR
					case "initiator":
						return INITIATOR
					}

				}
			}
		}
	}
	return funcType
}

func (method *Function) Parse() bool {
	funcType := method.parseComment()

	switch funcType {
	case CREATOR:
		returnStruct := method.parseCreator()
		if returnStruct != nil {
			method.funcManager.addCreator(returnStruct, method)
		}
	case INITIATOR:
		//后面如果需要添加inititor排序，需要新建函数返回Initiator
		method.parseCreator()
		method.funcManager.addInitiator(method)
		// &Variable{
		// 	name:    method.Results[0].name,
		// 	creator: method,
		// 	class:   returnStruct,
		// })
	case SERVLET:
		method.parseServlet()
		method.funcManager.addServlet(method)
	}
	return true
}

func (method *Function) parseCreator() *Struct {
	funcDecl := method.function
	returnTypeList := funcDecl.Type.Results.List
	if len(returnTypeList) != 1 {
		log.Fatalf("creator %s should have one return value", method.Name)
	}
	// 1. 返回其他包的是*ast.SelectorExpr; 返回本包的是什么？
	// 2. 如何区分返回的是指针还是结构体
	structType := method.parseFieldType(returnTypeList[0])
	if structType != nil {
		struct1 := structType.class
		method.Results = append(method.Results, structType)
		return struct1
	} else {
		log.Fatalf("creator %s has unknow type %V\n", method.Name, returnTypeList[0].Type)
	}
	return nil
}
func (method *Function) parseServlet() {
	funcDecl := method.function
	paramsList := funcDecl.Type.Params.List
	if len(paramsList) < 2 {
		log.Fatalf("servlet %s should have at least two parameters", method.Name)
	}
	structType := method.parseFieldType(paramsList[0])
	// 仅关心第一个参数；
	// 暂时没有关心返回值
	method.Params = append(method.Params, structType)
}

// 解析参数或者返回值的一个变量
func (method *Function) parseFieldType(field *ast.Field) *Variable {
	var selectorExpr *ast.SelectorExpr
	var isPointer bool
	if fieldType, ok := field.Type.(*ast.StarExpr); ok {
		if selectorExpr, ok = fieldType.X.(*ast.SelectorExpr); !ok {
			fmt.Printf("function %s has unknow type %V\n", method.Name, field.Type)
			return nil
		}
		isPointer = true
	} else if fieldType, ok := field.Type.(*ast.SelectorExpr); ok {
		isPointer = false
		selectorExpr = fieldType
	} else {
		fmt.Printf("function %s has unknow type %V\n", method.Name, field.Type)
		return nil
	}
	// 此处有三种情况
	// 1. 返回一个本项目存在结构体，mymode.Struct
	// 2. 返回一个本pkg的结构体，Struct
	// 3. 返回一个第三方的结构体体
	modelName := selectorExpr.X.(*ast.Ident).Name
	structName := selectorExpr.Sel.Name
	pkgPath := method.goFile.getImportPath(modelName, method.Name)
	pkg := method.goFile.pkg.Project.getPackage(pkgPath, true)
	var nameOfReturn0 string
	switch len(field.Names) {
	case 0:
		nameOfReturn0 = ""
	case 1:
		nameOfReturn0 = field.Names[0].Name
	default:
		log.Fatalf("initiator %s should have one or null in %s return value", method.Name, method.goFile.path)
	}
	struct1 := pkg.getStruct(structName, true)
	return &Variable{
		name:      nameOfReturn0,
		class:     struct1,
		isPointer: isPointer,
		// creator:   method,
	}
}

// 产生本方法即成到路由中去的方法
// file: 表示在那个文件中产生；
// receiverPrefix用于记录调用函数的receiver，仅有当Method时才用到，否则为空；
func (method *Function) GenerateCode(file *GenedFile, receiverPrefix string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	// file.getImport(method.pkg.Project.getModePath("basic"), "basic")
	codeFmt := `
	router.POST("%s", func(c *gin.Context) {
		%s
		if err := c.ShouldBindJSON(request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		response, err := %s%s(request, c)
		c.JSON(200, Response{
			Object:  response,
			Code:    err.Code,
			Message: err.Message,
		})
	})
	`
	var variableCode string
	variable := *method.Params[0]
	variable.name = "request"
	// 从receiver中查找是否有Creator方法
	creator := method.funcManager.getCreator(variable.class)
	if creator != nil {
		variable.creator = creator
		variable.isPointer = creator.Results[0].isPointer
	} else {
		variable.isPointer = true
	}
	if variable.isPointer {
		variableCode = "request:=" + variable.generateCode(receiverPrefix, file) + "\n"
	} else {
		variableCode = "requestObj:=" + variable.generateCode(receiverPrefix, file) + "\n request:=&requestObj\n"
	}

	return fmt.Sprintf(codeFmt,
		method.Url,
		variableCode,
		receiverPrefix, method.Name,
	)
}
