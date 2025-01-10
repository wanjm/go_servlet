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
	initiators []*DependNode         //初始化函数依赖关系
	servlets   []*Function           //记录路由代码
	postAction map[string]*Function  //记录后置操作
}

func createFunctionManager() FunctionManager {
	return FunctionManager{
		creators:   make(map[*Struct]*Function),
		postAction: make(map[string]*Function),
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
	funcManager.initiators = append(
		funcManager.initiators,
		&DependNode{
			function: initiator,
		},
	)
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
	serverName   string // server group name
	Url          string // url
	title        string // 函数描述，供swagger使用
	method       string // http方法，GET,POST，默认是POST
	isDeprecated bool
	funcType     int //函数类型，filter，servlet，websocket，prpc，initiator,creator
	security     []string
}

func (comment *functionComment) dealValuePair(key, value string) {
	key = strings.ToLower(key)
	switch key {
	case Url:
		comment.Url = value
		if comment.funcType == NOUSAGE {
			//默认是servlet
			comment.funcType = SERVLET
			if len(comment.method) == 0 {
				comment.method = POST
			}
		}
	case Creator:
		comment.funcType = CREATOR
	case UrlFilter:
		comment.Url = value
		comment.funcType = FILTER
	case FilterConst:
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
	case Security:
		comment.security = strings.Split(value, ",")
	case ConstMethod:
		comment.method = strings.ToUpper(value)
	case Title:
		comment.title = strings.Trim(value, "\"")
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

func createFunction(f *ast.FuncDecl, goFile *GoFile, funcManager *FunctionManager) *Function {
	return &Function{
		function:    f,
		Name:        f.Name.Name,
		pkg:         goFile.pkg,
		goFile:      goFile,
		funcManager: funcManager,
	}
}

func (method *Function) Parse() bool {
	parseComment(method.function.Doc, &method.comment)
	// 跳过不感兴趣的Func；
	if method.comment.funcType == NOUSAGE {
		if strings.HasPrefix(method.Name, "PostAction") {
			method.funcManager.postAction[method.Name[10:]] = method
		}
		return true
	}
	method.parseParameter(method.function.Type)
	switch method.comment.funcType {
	case CREATOR:
		//当将来有Creator方法返回为interface是，此处的findStruct(true)需要修改
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
		for _, name := range param.Names {
			nfield := field
			nfield.name = name.Name
			method.Params = append(method.Params, &nfield)
			break
		}
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
	results := funcDecl.Type.Results
	if results == nil {
		return nil
	}
	returnTypeList := results.List
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
	// sb.WriteString(method.genTraceId(file))
	sb.WriteString(receiverPrefix + method.Name + "(c,c.Writer,c.Request)\n")
	sb.WriteString("})\n")
	return sb.String()
}
func (method *Function) genTraceId(file *GenedFile) string {
	file.getImport("github.com/rs/xid", "xid")
	file.getImport("context", "context")
	return `
	var Request = c.Request
	tid := xid.New().String()
	c.Request = Request.WithContext(context.WithValue(Request.Context(), TraceIdNameInContext, tid))
	`
}

func (method *Function) GenerateRpcServlet(file *GenedFile, receiverPrefix string) string {
	file.getImport("github.com/gin-gonic/gin", "gin")
	file.getImport("context", "context")
	file.getImport("github.com/rs/xid", "xid")
	var sb strings.Builder
	sb.WriteString("router.POST(" + method.comment.Url + ", func(c *gin.Context) {\n")
	var interfaceArgs string
	var realParams string
	for i := 1; i < len(method.Params); i++ {
		param := method.Params[i]
		name := fmt.Sprintf("arg%d", i)
		variable := Variable{
			isPointer: param.isPointer,
			class:     param.findStruct(true),
			name:      "request",
		}
		sb.WriteString(name + ":=" + variable.generateCode(receiverPrefix, file) + "\n")
		if !param.isPointer {
			interfaceArgs += "&" + name + ","
		} else {
			interfaceArgs += name + ","
		}
		realParams += "," + name
	}

	sb.WriteString(fmt.Sprintf("var request=[]interface{}{%s}\n", interfaceArgs))
	sb.WriteString(`if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(200, map[string]interface{}{
			"o": []any{&Error{Code: 4, Message: err.Error()}},
			"c": 0,
		})
		return
	}
	var Request= c.Request;
	tid := Request.Header.Get(TraceId)
	if len(tid) <=0 {
		tid = xid.New().String()
	}
	c.Request = Request.WithContext(context.WithValue(Request.Context(), TraceIdNameInContext, tid))
	`)
	var objString string
	var objResult string
	// 返回值仅有一个是Error；
	if len(method.Results) == 2 {
		objResult = "response,"
		objString = "\"o\":[]any{[code,response},"
	} else {
		objString = "\"o\":[]any{code},"
	}
	// 返回值有两个，一个是response，一个是Error；
	// 代码暂不检查是否超过两个；
	//${objResult} err:= ${receiverPrefix}${method.Name}(c${realParams}
	sb.WriteString(fmt.Sprintf(`%s err := %s%s(c%s)
		var code any
		if err.Code != 0 {
			code = &Error{Code: int(err.Code), Message: err.Message}
		}
		c.JSON(200, map[string]interface{}{
			%s
			"c":    0,
		})
	`, objResult, receiverPrefix, method.Name, realParams, objString))
	sb.WriteString("})\n") //end of router.POST
	return sb.String()
}

// 产生postCall的部分代码
func (method *Function) generateServletPostCall(file *GenedFile, sb *strings.Builder) {
	file.getImport("github.com/gin-gonic/gin", "gin")
	sb.WriteString("router." + method.comment.method + "(" + method.comment.Url)
	var server = method.pkg.Project.servers[method.comment.serverName]
	methodUrl := method.comment.Url
	for _, filter := range server.urlFilters {
		filterUrl := filter.url
		if len(filterUrl) > 0 && strings.Contains(methodUrl, filterUrl) {
			sb.WriteString(",")
			sb.WriteString(filter.genName)
			break
		}
	}
	sb.WriteString(", func(c *gin.Context) {\n")
}

// 产生本方法即成到路由中去的方法
// file: 表示在那个文件中产生；
// receiverPrefix用于记录调用函数的receiver，仅有当Method时才用到，否则为空；
func (method *Function) GenerateServlet(file *GenedFile, receiverPrefix string) string {
	var sb strings.Builder
	method.generateServletPostCall(file, &sb)
	var realParams string
	var rawServlet = false
	//  有request请求，需要解析request，有些情况下，服务端不需要request；
	if len(method.Params) >= 2 {
		var variableCode string
		methodUrl := strings.Trim(method.comment.Url, "\"")
		paramIndex := 1
		requestParam := method.Params[paramIndex]
		realParams += "," + requestParam.name
		if strings.Contains(methodUrl, ":") {
			variableCode = requestParam.name + ":=" + requestParam.generateCode(receiverPrefix, file) + "\n"
			sb.WriteString(variableCode)
			names := strings.Split(methodUrl, "/")
			for _, name := range names {
				if strings.Contains(name, ":") {
					sb.WriteString(fmt.Sprintf("%s.%s=c.Param(\"%s\")\n", requestParam.name, name[1:], name[1:]))
				}
			}
			paramIndex = 2
		}
		requestParam = method.Params[paramIndex]
		if requestParam.pkg.modPath == "net/http" {
			// 此处是临时添加的解决第三方回调的问题；
			// 具体如何识别，1. 参数直接使用http.Request；
			// 2. 返回值改为basic.HTTPError.(Code表示http头的code，response就是完整的报文体) 这个更好。这样第一条就可以是根据需要传入
			realParams += ",c.Request"
			rawServlet = true
		} else {
			variableCode = "request:=" + requestParam.generateCode(receiverPrefix, file) + "\n"
			sb.WriteString(variableCode)
			sb.WriteString(`
		// 利用gin的自动绑定功能，将请求内容绑定到request对象上；兼容get,post等情况
		if err := c.ShouldBind(request); err != nil {
			c.JSON(200, Response{
			Code: 4,
			Message: "param error",
			})
			return
		}
		`)
			realParams += ",request"
		}
	}
	var objString string
	var objName string
	var objResult string
	// 返回值仅有一个是Error；
	if len(method.Results) == 2 {
		objName = "response"
		objResult = objName + ","
		objString = "Object:response,"
	}
	// sb.WriteString(method.genTraceId(file))
	// 返回值有两个，一个是response，一个是Error；
	// 代码暂不检查是否超过两个；
	sb.WriteString(fmt.Sprintf("%s err := %s%s(c%s)\n", objResult, receiverPrefix, method.Name, realParams))
	//realParams后续考虑使用strings.Join()来处理；潜力基本挖光了
	//此处后续考虑解析参数格式，然后添加正确的写入顺序
	if postAction, ok := method.funcManager.postAction[method.Name]; ok {
		sb.WriteString(fmt.Sprintf("%s%s(c%s,%serr)\n", receiverPrefix, postAction.Name, realParams, objResult))
	}
	if rawServlet {
		sb.WriteString("c.JSON(int(err.Code), response)")
	} else {
		sb.WriteString("var code=200;\n")
		if method.comment.method == "GET" {
			sb.WriteString(`
		if err.Code==500 {
			// 临时兼容health check;
			code=500
		}
	`)
		}
		if len(objName) > 0 {
			var a = *method.Results[0]
			a.name = objName
			if a.isPointer {
				panic("response should not be pointer in " + receiverPrefix + method.Name)
			}
			a.genCheckArrayNil("", file, &sb)
		}
		sb.WriteString(fmt.Sprintf(`
		c.JSON(code, Response{
			%s
			Code:   int(err.Code),
			Message: err.Message,
		})
	`, objString))
	}
	sb.WriteString("})\n")

	return sb.String()
}

// 生成调用本函数的代码
func (creator *Function) genCallCode(receiverPrefix string, file *GenedFile) string {
	var prefix string
	if len(receiverPrefix) > 0 {
		prefix = receiverPrefix
	} else {
		pkg := creator.pkg
		impt := file.getImport(pkg.modPath, pkg.modName)
		prefix = impt.Name + "."
	}
	var paramstr = make([]string, len(creator.Params))
	for i, param := range creator.Params {
		paramstr[i] = param.generateCode(prefix, file)
	}
	return prefix + creator.Name + "(" + strings.Join(paramstr, ",") + ")"
}
