package astinfo

import (
	"go/ast"
	"strings"
)

type Method struct {
	Name      string // method name
	Params    string // method params （input）
	Results   string // method results（output)
	Url       string // method url from comments;
	HasCreate bool   // has create method 返回值同Params
}

func (method *Method) InitFromFunc(f *ast.FuncDecl) {
	method.Name = f.Name.Name
	method.initFromComment(f)
}
func (method *Method) initFromComment(f *ast.FuncDecl) {
	if f.Doc == nil {
		return
	}
	for _, comment := range f.Doc.List {
		if comment.Text == "" {
			continue
		}
		text := strings.Trim(comment.Text, "/ \t") // 去掉前后的空格和斜杠
		text = strings.ReplaceAll(text, "\t ", "")
		if strings.HasPrefix(text, "@url=") {
			method.Url = text[5:]
		}
	}
}
