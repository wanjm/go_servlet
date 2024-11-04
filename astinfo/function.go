package astinfo

import (
	"fmt"
	"go/ast"
	"log"
	"strings"
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

const (
	POST = "POST"
	GET  = "GET"
)

// @goservlet url="/test" filter=[prpc|servlet|""]; creator;initiator;websocket;
type functionComment struct {
	serverName   string
	Url          string
	method       string
	isDeprecated bool
	funcType     int
}

func (comment *functionComment) dealValuePair(key, value string) {
	switch key {
	case Url:
		comment.Url = value
		if comment.funcType == NOUSAGE {
			//默认是servlet
			comment.funcType = SERVLET
		}
	case Creator:
		comment.funcType = CREATOR
	case UrlFilter:
		comment.Url = value
		comment.funcType = FILTER
	case Filter:
		if len(value) == 0 {
			value = Servlet
		}
		comment.serverName = value
		comment.funcType = FILTER
	case Servlet:
		comment.serverName = value
		comment.funcType = SERVLET
	case Prpc:
		comment.serverName = value
		comment.funcType = PRPC
	case Initiator:
		comment.funcType = INITIATOR
	case Websocket:
		comment.method = GET
		comment.funcType = WEBSOCKET
	default:
		fmt.Printf("unknown key '%s' in function comment\n", key)
	}
}

type Function struct {
	Name        string   // method name
	Params      []*Field // method params, 下标0是request
	Results     []*Field // method results（output)
	function    *ast.FuncDecl
	pkg         *Package
	goFile      *GoFile
	funcManager *FunctionManager
	comment     functionComment
	// Url        string // method url from comments;
	// deprecated bool
}

func createFunction(f *ast.FuncDecl, goFile *GoFile) *Function {
	return &Function{
		function:    f,
		Name:        f.Name.Name,
		pkg:         goFile.pkg,
		goFile:      goFile,
		funcManager: &goFile.pkg.FunctionManager,
	}
}

func (method *Function) Parse() bool {
	parseComment(method.function.Doc, &method.comment)
	method.parseParameter(method.function.Type)
	switch method.comment.funcType {
	case CREATOR:
		//当将来有Creator方法返回位interface是，此处的findStruct(true)需要修改
		method.parseCreator()
		returnStruct := method.Results[0].findStruct(true)
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
	case SERVLET, WEBSOCKET, PRPC:
		method.funcManager.addServlet(method)
	case FILTER:
		method.pkg.Project.addUrlFilter(method, method.comment.serverName)
	}
	return true
}

// 解析参数和返回值
func (method *Function) parseParameter(paramType *ast.FuncType) bool {
	for _, param := range paramType.Params.List {
		field := Field{
			ownerInfo: "function Name is " + method.Name,
		}
		field.parseType(param.Type, method.goFile)
		//此处可能多个参数 a,b string的格式暂时仅处理一个；
		if len(param.Names) > 1 {
			log.Fatalf("function %s has more than one parameter", method.Name)
		}
		if len(param.Names) > 0 {
			field.name = param.Names[0].Name
		}
		method.Params = append(method.Params, &field)
	}
	if paramType.Results != nil {
		for _, result := range paramType.Results.List {
			field := Field{
				ownerInfo: "function Name is " + method.Name,
			}
			field.parseType(result.Type, method.goFile)

			if len(result.Names) != 0 {
				field.name = result.Names[0].Name
			}
			method.Results = append(method.Results, &field)
		}
	}
	return true
}

func (method *Function) parseCreator() *Struct {
	funcDecl := method.function
	returnTypeList := funcDecl.Type.Results.List
	if len(returnTypeList) != 1 {
		log.Fatalf("creator %s should have one return value", method.Name)
	}
	return nil
}

func (method *Function) parseServlet() {
	funcDecl := method.function
	paramsList := funcDecl.Type.Params.List
	if len(paramsList) < 2 {
		// 	log.Fatalf("servlet %s should have at least two parameters", method.Name)
	}
}

func (method *Function) GenerateWebsocket(file *GenedFile, receiverPrefix string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	var sb = strings.Builder{}
	sb.WriteString("router.GET(" + method.comment.Url + ", func(c *gin.Context) {\n")
	sb.WriteString(receiverPrefix + method.Name + "(c,c.Writer,c.Request)\n")
	sb.WriteString("})\n")
	return sb.String()
}

func (method *Function) GenerateRpcServlet(file *GenedFile, receiverPrefix string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	var sb strings.Builder
	sb.WriteString("router.POST(" + method.comment.Url + ", func(c *gin.Context) {\n")
	var interfaceArgs string
	var realParams string
	for i := 1; i < len(method.Params); i++ {
		param := method.Params[i]
		name := fmt.Sprintf("arg%d", i)
		sb.WriteString("var " + name + " " + param.class.(*Struct).Name + "\n")
		interfaceArgs += "&" + name + ","
		realParams += "," + name
	}

	sb.WriteString(fmt.Sprintf("var request=[]interface{}{%s}\n", interfaceArgs))
	sb.WriteString(`if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	`)
	sb.WriteString(fmt.Sprintf(`response, err := %s%s(c%s)
		c.JSON(200, map[string]interface{}{
			"c":err.Code,
			"o":response,
		})
	`, receiverPrefix, method.Name, realParams))
	sb.WriteString("})\n") //end of router.POST
	return sb.String()
}

// 产生本方法即成到路由中去的方法
// file: 表示在那个文件中产生；
// receiverPrefix用于记录调用函数的receiver，仅有当Method时才用到，否则为空；
func (method *Function) GenerateServlet(file *GenedFile, receiverPrefix string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	var sb strings.Builder
	sb.WriteString("router.POST(" + method.comment.Url + ", func(c *gin.Context) {\n")
	var requestName string
	//  有request请求，需要解析request，有些情况下，服务端不需要request；
	if len(method.Params) >= 2 {
		var variableCode string
		requestParam := method.Params[1]
		variable := Variable{
			isPointer: requestParam.isPointer,
			class:     requestParam.findStruct(true),
			name:      "request",
		}
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
		sb.WriteString(variableCode)
		sb.WriteString(`if err := c.ShouldBindJSON(request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		`)
		requestName = ",request"
	}
	var objString string
	var objResult string
	// 返回值仅有一个是Error；
	if len(method.Results) == 2 {
		objResult = "response,"
		objString = "Object:response,"
	}
	// 返回值有两个，一个是response，一个是Error；
	// 代码暂不检查是否超过两个；
	sb.WriteString(fmt.Sprintf(`%s err := %s%s(c%s)
		c.JSON(200, Response{
			%s
			Code:    err.Code,
			Message: err.Message,
		})
	`, objResult, receiverPrefix, method.Name, requestName, objString))
	sb.WriteString("})\n")

	return sb.String()
}
