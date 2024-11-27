package astinfo

import (
	"go/ast"
)

type Method struct {
	Receiver *Struct
	*Function
	Url       string // method url from comments;
	HasCreate bool   // has create method 返回值同Params
}

func createMethod(f *ast.FuncDecl, goFile *GoFile) *Method {
	recvType := f.Recv.List[0].Type
	var nameIndent *ast.Ident
	if starExpr, ok := recvType.(*ast.StarExpr); ok {
		nameIndent = starExpr.X.(*ast.Ident)
	} else {
		nameIndent = recvType.(*ast.Ident)
	}
	// 由于代码的位置关系，这一步不一定会找到，所以自己创建了。
	receiver := goFile.pkg.getStruct(nameIndent.Name, true)
	function := createFunction(f, goFile, &receiver.FunctionManager)
	function.comment.serverName = receiver.comment.groupName

	return &Method{
		Receiver: receiver,
		Function: function,
	}
}
