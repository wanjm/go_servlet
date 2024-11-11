package astinfo

import (
	"go/ast"
)

type Method struct {
	Receiver *Struct
	Function
	Url       string // method url from comments;
	HasCreate bool   // has create method 返回值同Params
}

func createMethod(f *ast.FuncDecl, goFile *GoFile) *Method {
	function := createFunction(f, goFile)
	return &Method{
		Function: *function,
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
	method.funcManager = &method.Receiver.FunctionManager
	method.Function.Parse()
	method.Function.comment.serverName = method.Receiver.comment.groupName
	return true
}
