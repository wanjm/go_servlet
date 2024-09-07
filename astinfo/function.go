package astinfo

import (
	"fmt"
	"go/ast"
	"log"
	"os"
	"regexp"
	"strings"
)

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
)

type FunctionManag interface {
	addServlet(*Function)
	addCreator(childClass *Struct, method *Function)
}

type FunctionManager struct {
	servletMethods []*Function           //记录路由代码
	creatorMethods map[*Struct]*Function //纪录构建默认参数的代码, key是构建的struct
}

func (funcManager *FunctionManager) addServlet(function *Function) {
	funcManager.servletMethods = append(funcManager.servletMethods, function)
}

func (funcManager *FunctionManager) addCreator(childClass *Struct, function *Function) {
	funcManager.creatorMethods[childClass] = function
}
func (funcManager *FunctionManager) getCreator(childClass *Struct) (function *Function) {
	return funcManager.creatorMethods[childClass]
}

type Function struct {
	Name        string      // method name
	Params      []*Variable // method params, 下标0是request
	Results     []*Variable // method results（output)
	function    *ast.FuncDecl
	pkg         *Package
	Url         string // method url from comments;
	deprecated  bool
	goFile      *GoFile
	funcManager *FunctionManager
}

func createFunction(f *ast.FuncDecl, goFile *GoFile) *Function {
	return &Function{
		function: f,
		pkg:      goFile.pkg,
		goFile:   goFile,
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
	modelName := selectorExpr.X.(*ast.Ident).Name
	structName := selectorExpr.Sel.Name
	pkgPath := method.goFile.Imports[modelName]
	if len(pkgPath) == 0 {
		fmt.Printf("failed to find package %s in %s\n", modelName, method.Name)
		os.Exit(1)
	}
	pkg := method.goFile.pkg.Project.getPackage(pkgPath, true)
	struct1 := pkg.getStruct(structName, true)
	return &Variable{
		class:     struct1,
		isPointer: isPointer,
	}
}

// 产生本方法即成到路由中去的方法
func (method *Function) GenerateCode(file *GenedFile, receiverName string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	file.getImport(method.pkg.Project.getModePath("basic"), "basic")
	codeFmt := `
	router.POST("%s", func(c *gin.Context) {
		%s
		if err := c.ShouldBindJSON(request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		response, err := %s.%s(request, c)
		c.JSON(200, basic.Response{
			Object:  response,
			Code:    err.Code,
			Message: err.Message,
		})
	})
	`
	var variableCode string
	variable := *method.Params[0]
	variable.calledInFile = file
	variable.name = "request"
	// 从receiver中查找是否有Creator方法
	creator := method.funcManager.getCreator(variable.class)
	if creator != nil {
		variable.creator = creator
		variable.isPointer = creator.Results[0].isPointer
	}
	if variable.isPointer {
		variableCode = "request:=" + variable.generateCode(receiverName+".") + "\n"
	} else {
		variableCode = "requestObj:=" + variable.generateCode(receiverName+".") + "\n request:=&requestObj\n"
	}

	return fmt.Sprintf(codeFmt,
		method.Url,
		variableCode,
		receiverName, method.Name,
	)
}
