package astinfo

import (
	"fmt"
	"go/ast"
	"log"
	"os"
	"strings"
)

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
)

type Function struct {
	Name     string      // method name
	Params   []*Variable // method params, 下标0是request
	Results  []*Variable // method results（output)
	function *ast.FuncDecl
	pkg      *Package
}

type Method struct {
	Receiver *Struct
	Function
	Url       string // method url from comments;
	HasCreate bool   // has create method 返回值同Params
	goFile    *GoFile
}

func createMethod(f *ast.FuncDecl, goFile *GoFile) *Method {
	return &Method{
		Function: Function{
			function: f,
			pkg:      goFile.pkg,
		},
		goFile: goFile,
	}
}
func (method *Method) Parse() bool {
	f := method.function
	recvType := f.Recv.List[0].Type
	var nameIndent *ast.Ident
	if starExpr, ok := recvType.(*ast.StarExpr); ok {
		nameIndent = starExpr.X.(*ast.Ident)
	} else {
		nameIndent = recvType.(*ast.Ident)
	}
	method.Receiver = method.goFile.pkg.getStruct(nameIndent.Name, true)
	method.Name = f.Name.Name
	funcType := method.parseComment()

	switch funcType {
	case CREATOR:
		method.parseCreator()
	case SERVLET:
		method.parseServlet()
	}
	return true
}

func (method *Method) checkIfCreator() bool {
	// 暂时简化处理Creator方法
	return strings.HasSuffix(method.Name, "Creator")
}

// 解析参数,解析返回值
func (method *Method) initParamether() {
	// params := f.Type.Params
}
func (method *Method) parseServlet() {
	funcDecl := method.function
	paramsList := funcDecl.Type.Params.List
	if len(paramsList) < 2 {
		log.Fatalf("servlet %s of %s should have at least two parameters", method.Name, method.Receiver.Name)
	}
	structType := method.parseFieldType(paramsList[0])
	// 仅关心第一个参数；
	// 暂时没有关心返回值
	method.Params = append(method.Params, structType)
	method.Receiver.addServlet(method)
}

// 解析参数或者返回值的一个变量
func (method *Method) parseFieldType(field *ast.Field) *Variable {
	var selectorExpr *ast.SelectorExpr
	var isPointer bool
	if fieldType, ok := field.Type.(*ast.StarExpr); ok {
		if selectorExpr, ok = fieldType.X.(*ast.SelectorExpr); !ok {
			fmt.Printf("function %s::%s has unknow type %V\n", method.Receiver.Name, method.Name, field.Type)
			return nil
		}
		isPointer = true
	} else if fieldType, ok := field.Type.(*ast.SelectorExpr); ok {
		isPointer = false
		selectorExpr = fieldType
	} else {
		fmt.Printf("function %s::%s has unknow type %V\n", method.Receiver.Name, method.Name, field.Type)
		return nil
	}
	modelName := selectorExpr.X.(*ast.Ident).Name
	structName := selectorExpr.Sel.Name
	pkgPath := method.goFile.Imports[modelName]
	if len(pkgPath) == 0 {
		fmt.Printf("failed to find package %s in %s::%s\n", modelName, method.Receiver.Name, method.Name)
		os.Exit(1)
	}
	pkg := method.goFile.pkg.Project.getPackage(pkgPath, true)
	struct1 := pkg.getStruct(structName, true)
	return &Variable{
		class:     struct1,
		isPointer: isPointer,
	}
}
func (method *Method) parseCreator() {
	funcDecl := method.function
	returnTypeList := funcDecl.Type.Results.List
	if len(returnTypeList) != 1 {
		log.Fatalf("creator %s of %s should have one return value", method.Name, method.Receiver.Name)
	}
	// 1. 返回其他包的是*ast.SelectorExpr; 返回本包的是什么？
	// 2. 如何区分返回的是指针还是结构体
	structType := method.parseFieldType(returnTypeList[0])
	if structType != nil {
		struct1 := structType.class
		method.Receiver.addCreator(struct1, method)
		method.Results = append(method.Results, structType)
	} else {
		log.Fatalf("creator %s of %s has unknow type %V\n", method.Name, method.Receiver.Name, returnTypeList[0].Type)
	}
}

// 解析注释
func (method *Method) parseComment() int {
	f := method.function
	method.Name = f.Name.Name
	funcType := NOUSAGE
	// isCreator := strings.HasSuffix(method.Name, "Creator")
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			text := strings.Trim(comment.Text, "/ \t") // 去掉前后的空格和斜杠
			text = strings.ReplaceAll(text, "\t ", "")
			if strings.HasPrefix(text, "@url=") {
				method.Url = strings.Trim(text[5:], "\"'")
				funcType = SERVLET
			} else if text == "@creator" {
				funcType = CREATOR
			}
		}
	}
	if funcType == NOUSAGE {
		if method.checkIfCreator() {
			funcType = CREATOR
		}
	}
	return funcType
}

// 产生本方法即成到路由中去的方法
func (method *Method) GenerateCode(file *GenedFile) string {
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
	if variable.isPointer {
		variableCode = "request:=" + variable.generateCode() + "\n"
	} else {
		variableCode = "requestObj:=" + variable.generateCode() + "\n request:=&requestObj\n"
	}

	return fmt.Sprintf(codeFmt,
		method.Url,
		variableCode,
		method.Receiver.receiver.name, method.Name,
	)
}

// func (m *Method) generateCrateor() string {
// 	return m.Receiver.GetCreatorCode4Struct(m.Params[0].class)
// }
