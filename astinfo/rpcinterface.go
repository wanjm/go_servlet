package astinfo

import (
	"go/ast"
	"strings"
)

// RpcInterface 代表一个rpc接口，该接口会在GoFile::parseType中被解析出来，放置到对应的Package中
type RpcInterface struct {
	Name           string
	ClientVariable *Field
	Package        *Package
	structName     string //生成时是结构体的名字
	FunctionManager
}

func CreateRpcInterface(name string, pkg *Package) *RpcInterface {
	return &RpcInterface{
		Name:            name,
		Package:         pkg,
		FunctionManager: createFunctionManager(),
	}
}

func (rpcInterface *RpcInterface) Parse(astInterface *ast.InterfaceType, goFile *GoFile) {
	// 接口的method的名字是变量名
	for _, method := range astInterface.Methods.List {
		function := Function{
			Name:   method.Names[0].Name,
			goFile: goFile,
		}
		function.parseParameter(method.Type.(*ast.FuncType))
		rpcInterface.addServlet(&function)
		_ = method
	}
}

func (class *RpcInterface) GenerateCode(file *GenedFile, sb *strings.Builder) {
	class.structName = class.Name + "Struct"
	sb.WriteString("type " + class.structName + " struct {\nclient RpcClient\n}\n")

	// 生成rpc strutct 代码；
	for _, servlet := range class.servlets {
		class.genRpcClientCode(servlet, sb)
	}
}

func (rpcInterface *RpcInterface) genRpcClientCode(method *Function, sb *strings.Builder) {
	// func (jsinternal *JsinternalStruct) GetTokenDetail(tokenStr string) (obj *basic.TokenUser, err error) {
	// 	var argument = []interface{}{tokenStr}
	// 	var res = jsinternal.client.SendRequest("/token/getDetail", argument)
	// 	if res.C != 0 {
	// 		return nil, nil
	// 	}
	// 	json.Unmarshal(res.O, &obj)
	// 	return &obj, nil
	// }
	sb.WriteString("func (receiver *" + rpcInterface.structName + ") " + method.Name + "(")
	// 定义入参
	var args []string
	var params []string
	for _, param := range method.Params {
		info := param.name + " "
		if param.isPointer {
			info += "*"
		}
		info += param.class.Name
		params = append(params, info)
		args = append(args, param.name)
	}
	sb.WriteString(strings.Join(params, ","))
	sb.WriteString(")(")
	//定义返回值
	var results []string
	var resultP0 = method.Results[0]
	info := "obj*" + resultP0.name
	results = append(results, info)
	info = "err error"
	results = append(results, info)
	sb.WriteString(strings.Join(results, ","))
	// 定义函数结束
	sb.WriteString("){\n")

	// 生成远程参数
	sb.WriteString("var argument = []interface{}{")
	sb.WriteString(strings.Join(args, ","))
	sb.WriteString("}\n")

	// 生成调用代码
	sb.WriteString("var res = receiver.client.SendRequest(")
	sb.WriteString(method.Url)
	sb.WriteString(", argument)\n")

	sb.WriteString(`
	if res.C != 0 {
		return nil, nil
	}
	json.Unmarshal(res.O, obj)
	return obj, nil
}`)
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

func (client *RpcClient) SendRequest(name string, array []interface{}) RpcResult {
	content, marError := json.Marshal(array)
	if marError != nil {
		fmt.Printf("%v\n", marError)
		return RpcResult{C: 1, O: nil}
	}
	resp, error := http.Post(client.Prefix+name, "", bytes.NewReader(content))
	if error != nil {
		fmt.Printf("%v\n", error)
	}
	var res = RpcResult{}
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&res)
	return res
}
`
}
