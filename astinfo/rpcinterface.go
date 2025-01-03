package astinfo

import (
	"fmt"
	"go/ast"
	"strings"
)

// @ goservlet rpcClient host=basic.Cfg.Rpc.JsinternalHost port=basic.Cfg.Rpc.JsinternalPort
type RpcInterfaceConfig struct {
	IsRpcClient bool
	Host        string
	Port        string
}

func (config *RpcInterfaceConfig) dealValuePair(key, value string) {
	switch key {
	case "host":
		config.Host = value
	case "rpcClient":
		config.IsRpcClient = true
	default:
		fmt.Printf("unkonw key value pair => key=%s,value=%s\n", key, value)
	}
}

// Interface 代表一个rpc接口，该接口会在GoFile::parseType中被解析出来，放置到对应的Package中
type Interface struct {
	config         *RpcInterfaceConfig
	Name           string
	ClientVariable *Field
	Package        *Package
	structName     string //生成时是结构体的名字
	FunctionManager
}

func CreateInterface(name string, pkg *Package) *Interface {
	return &Interface{
		Name:            name,
		Package:         pkg,
		FunctionManager: createFunctionManager(),
	}
}

// 目前仅解析注释中的rpcClient配置
func (rpcInterface *Interface) parseComment(doc *ast.CommentGroup) {
	var config = RpcInterfaceConfig{}
	parseComment(doc, &config)
	if config.IsRpcClient {
		// fmt.Printf("rpcInterface %s is rpcClient\n", rpcInterface.Name)
		rpcInterface.config = &config
	}
}
func (rpcInterface *Interface) Parse(astInterface *ast.InterfaceType, goFile *GoFile) {
	// 接口的method的名字是变量名
	for _, method := range astInterface.Methods.List {
		function := Function{
			Name:   method.Names[0].Name,
			goFile: goFile,
		}
		parseComment(method.Doc, &function.comment)
		function.parseParameter(method.Type.(*ast.FuncType))
		rpcInterface.addServlet(&function)
		_ = method
	}
}

func (class *Interface) GenerateCode(file *GenedFile, sb *strings.Builder) bool {
	// fmt.Printf("gen interface code servlet %d and conf=nil %v \n", len(class.servlets), (class.config == nil))
	if len(class.servlets) == 0 || class.config == nil {
		return false
	}
	class.structName = class.Name + "Struct"
	sb.WriteString("type " + class.structName + " struct {\nclient RpcClient\n}\n")

	// 生成rpc strutct 代码；
	for _, servlet := range class.servlets {
		class.genRpcClientCode(file, servlet, sb)
	}

	return true
}

func (rpcInterface *Interface) genRpcClientCode(file *GenedFile, method *Function, sb *strings.Builder) {
	// func (jsinternal *JsinternalStruct) GetTokenDetail(tokenStr string) (obj *basic.TokenUser, err error) {
	// 	var argument = []interface{}{tokenStr}
	// 	var res = jsinternal.client.SendRequest("/token/getDetail", argument)
	// 	if res.C != 0 {
	// 		return nil, nil
	// 	}
	// 	json.Unmarshal(res.O, &obj)
	// 	return &obj, nil
	// }
	file.getImport("context", "context")
	sb.WriteString("func (receiver *" + rpcInterface.structName + ") " + method.Name + "(ctx context.Context,")
	// 定义入参
	var args []string
	var params []string
	// 默认第一个是context
	for i, l := 1, len(method.Params); i < l; i++ {
		param := method.Params[i]
		info := param.name + " "
		if param.isPointer {
			info += "*"
		}
		if param.pkg != rpcInterface.Package.Project.rawPkg {
			oneImport := file.getImport(param.pkg.modPath, param.pkg.modName)
			info += oneImport.Name + "."
		}
		info += param.typeName
		params = append(params, info)
		args = append(args, param.name)
	}
	sb.WriteString(strings.Join(params, ","))
	sb.WriteString(")(")
	//定义返回值
	var results []string
	if len(method.Results) >= 2 {
		var resultP0 = method.Results[0]

		info := "obj "
		if resultP0.isPointer {
			info += "*"
		}
		typePkg := resultP0.pkg
		if typePkg != rpcInterface.Package.Project.rawPkg {
			oneImport := file.getImport(typePkg.modPath, typePkg.modName)
			info += oneImport.Name + "."
		}
		info += resultP0.typeName

		results = append(results, info)
	}
	info := "err error"
	results = append(results, info)
	sb.WriteString(strings.Join(results, ","))
	// 定义函数结束
	sb.WriteString("){\n")

	// 生成远程参数
	sb.WriteString("var argument = []interface{}{")
	sb.WriteString(strings.Join(args, ","))
	sb.WriteString("}\n")

	// 生成调用代码
	sb.WriteString("var res = receiver.client.SendRequest(ctx,")
	sb.WriteString(method.comment.Url)
	file.getImport("errors", "errors")
	file.getImport("context", "context")
	sb.WriteString(`, argument)
	if res.C != 0 {
		err=errors.New("failed to call`)
	sb.WriteString(method.Name)
	sb.WriteString(`")
		return
	}
	if res.O[0] != nil {
		err = res.O[0].(error)
		return 
	}
	`)
	if len(method.Results) >= 2 {
		sb.WriteString(`
		//无论object是否位指针，都需要取地址
		json.Unmarshal(*res.O[1].(*json.RawMessage), &obj)
		`)
		file.getImport("encoding/json", "json")
	}
	sb.WriteString("return\n}\n")
}

func GenerateRpcClientCode(file *GenedFile) string {
	return `
	package prpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RpcResult struct {
	C int             "json:\"c\""
	O json.RawMessage "json:\"o\""
}
type RpcClient struct {
	Prefix string
}

func (client *RpcClient) SendRequest(ctx context.Context, name string,  array []interface{}) RpcResult {
	content, marError := json.Marshal(array)
	if marError != nil {
		fmt.Printf("%v\n", marError)
		return RpcResult{C: 1, O: [2]interface{}{nil, json.RawMessage{}}}
	}
	req, err := http.NewRequest("POST", client.Prefix+name, bytes.NewReader(content))
	var resp *http.Response
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("TraceId", ctx.Value("TRID").(string))
		resp, err = http.DefaultClient.Do(req)
	}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	var res = RpcResult{
		O: [2]interface{}{&Error{}, &json.RawMessage{}},
	}
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&res)
	return res
}
`
}
