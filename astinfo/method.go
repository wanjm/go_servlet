package astinfo

import (
	"fmt"
	"go/ast"
	"strings"
)

const (
	NOUSAGE = iota
	CREATOR
	SERVLET
)

type Method struct {
	Receiver  *Struct
	Name      string    // method name
	Params    []*Struct // method params, 下标0是request
	Results   []*Struct // method results（output)
	Url       string    // method url from comments;
	HasCreate bool      // has create method 返回值同Params
}

func (method *Method) InitFromFunc(f *ast.FuncDecl) bool {
	method.Name = f.Name.Name
	method.initFromComment(f)
	return true
}

// 解析参数,解析返回值
func (method *Method) initParamether(f *ast.FuncDecl) {
	// params := f.Type.Params
}

// 解析注释
func (method *Method) initFromComment(f *ast.FuncDecl) int {
	method.Name = f.Name.Name
	funcType := NOUSAGE
	// isCreator := strings.HasSuffix(method.Name, "Creator")
	if f.Doc != nil {
		for _, comment := range f.Doc.List {
			text := strings.Trim(comment.Text, "/ \t") // 去掉前后的空格和斜杠
			text = strings.ReplaceAll(text, "\t ", "")
			if strings.HasPrefix(text, "@url=") {
				method.Url = strings.Trim(text[5:], "\"'")
				method.Receiver.addServlet(method)
				funcType = SERVLET
			} else if text == "@creator" {
				funcType = CREATOR
			}

		}
	}
	// 	if funcType!=NOUSAGE {
	// 	if funcType==CREATOR {
	// 		method.Receiver.addCreator(,method)
	// 	}
	// }
	return funcType
}

// 产生本方法即成到路由中去的方法
func (method *Method) GenerateCode() string {
	codeFmt := `
	router.POST("%s", func(c *gin.Context) {
		request := %s
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		response, err := %s.%s(&request, c)
		c.JSON(200, basic.Response{
			Object:  response,
			Code:    err.Code,
			Message: err.Message,
		})
	})
	`
	return fmt.Sprintf(codeFmt, method.Url, method.generateCrateor(), method.Receiver.variableName, method.Name)
}

func (m *Method) generateCrateor() string {
	return m.Receiver.GetCreatorCode4Struct(m.Params[0])
}
