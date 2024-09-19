package astinfo

import (
	"fmt"
	"go/ast"
	"log"
	"strings"
)

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
	INITIATOR
	URLFILTER
)
const TagPrefix = "@goservlet"
const GolangRawType = "rawType"
const UrlFilter = "urlfilter"
const Creator = "creator"
const Url = "url"
const Initiator = "initiator"

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
	Name        string   // method name
	Params      []*Field // method params, 下标0是request
	Results     []*Field // method results（output)
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
// 注释支持的格式为 @plaso url=xxx ; creator ; urlfilter=xxx
func (function *Function) parseComment() int {
	f := function.function
	function.Name = f.Name.Name
	funcType := NOUSAGE
	// isCreator := strings.HasSuffix(method.Name, "Creator")
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			text := strings.TrimLeft(comment.Text, "/ \t") // 去掉前面的空格和斜杠
			if strings.HasPrefix(text, TagPrefix) {
				newString := text[len(TagPrefix):]
				commands := strings.Split(newString, ";") // 多个参数以;分割
				for _, command := range commands {
					valuePair := strings.Split(command, "=") // 参数名和参数值以=分割
					valuePair[0] = strings.Trim(valuePair[0], " \t")
					if len(valuePair) == 2 {
						//去除前后空格和引号
						valuePair[1] = strings.Trim(valuePair[1], " \t\"'")
					}
					switch valuePair[0] {
					case Url:
						function.Url = valuePair[1]
						return SERVLET
					case Creator:
						return CREATOR
					case UrlFilter:
						function.Url = valuePair[1]
						return URLFILTER
					case Initiator:
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
	method.parseParameter(method.function.Type)
	switch funcType {
	case CREATOR:
		method.parseCreator()
		returnStruct := method.Results[0].class
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
	case URLFILTER:
		method.pkg.Project.addUrlFilter(method)
	}
	return true
}

// 解析参数和返回值
func (method *Function) parseParameter(paramType *ast.FuncType) bool {
	for _, param := range paramType.Params.List {
		field := Field{
			ownerInfo: "function Name is " + method.Name,
		}
		field.parse(param.Type, method.goFile)
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
			field.parse(result.Type, method.goFile)

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
		log.Fatalf("servlet %s should have at least two parameters", method.Name)
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
		response, err := %s%s(c, request)
		c.JSON(200, Response{
			Object:  response,
			Code:    err.Code,
			Message: err.Message,
		})
	})
	`
	var variableCode string
	requestParam := method.Params[1]
	variable := Variable{
		isPointer: requestParam.isPointer,
		class:     requestParam.class,
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

	return fmt.Sprintf(codeFmt,
		method.Url,
		variableCode,
		receiverPrefix, method.Name,
	)
}
