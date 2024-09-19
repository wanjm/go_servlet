package astinfo

import "go/ast"

type RpcInterface struct {
	Name           string
	ClientVariable *Field
	FunctionManager
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
