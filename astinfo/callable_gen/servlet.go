package callable_gen

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/wanjm/gos/astinfo"
)

type FilterInfo struct {
	FilterName    string
	FilterRawName string
	Func          *astinfo.Function
}
type ServletGen struct {
	filters       []*FilterInfo
	filterMap     map[string]*FilterInfo
	InternalError int
	DataError     int
}

func NewServletGen(dataError, internalError int) *ServletGen {
	servlet := &ServletGen{
		DataError:     dataError,
		InternalError: internalError,
		filterMap:     make(map[string]*FilterInfo),
	}
	return servlet
}
func (servlet *ServletGen) GetName() string {
	return "servlet"
}

var commongened bool

// 定义代码生成模板
const cJsonTemplate = `{{if .HasResponseKey}}
var responseKey {{.ImportName}}.{{.ResponseKey}}

type JsonString struct {
	context context.Context
	data    any
}

func (r JsonString) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	jsonBytes, err := json.Marshal(r.data)
	if err != nil {
		return err
	}
	v := r.context.Value(responseKey)
	if v != nil {
		*(v.(*string)) = string(jsonBytes)
	}
	_, err = w.Write(jsonBytes)
	return err
}

// WriteContentType (JSON) writes JSON ContentType.
func (r JsonString) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"application/json; charset=utf-8"}
	}
}

func cJSON(c *gin.Context, code int,response any) {
	c.Render(code, JsonString{
		context: c,
		data:    response,
	})
}
{{else}}
func cJSON(c *gin.Context, code int, response any) {
	c.JSON(code, response)
}
{{end}}

func getErrorCode(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	var errorCode int
	var errMessage = err.Error()
	if basicError,ok:=err.(Coder);ok{
		errorCode = basicError.GetErrorCode()
	}else{
		errorCode = 1
	}
	return errorCode, errMessage
}
type Coder interface {
	GetErrorCode() int
}
`

func (servlet *ServletGen) GenerateCommon(file *astinfo.GenedFile) {
	if commongened {
		return
	}
	commongened = true
	var content strings.Builder
	Project := astinfo.GlobalProject

	// 准备模板数据
	data := struct {
		HasResponseKey bool
		ImportName     string
		ResponseKey    string
	}{}

	if Project.Cfg.Generation.ResponseKey != "" {
		data.HasResponseKey = true
		oneImport := file.GetImport(astinfo.SimplePackage(Project.Cfg.Generation.ResponseMod, "xx"))
		data.ImportName = oneImport.Name
		data.ResponseKey = Project.Cfg.Generation.ResponseKey
		file.GetImport(astinfo.SimplePackage("context", "context"))
		file.GetImport(astinfo.SimplePackage("encoding/json", "json"))
		file.GetImport(astinfo.SimplePackage("net/http", "http"))
	}

	// 解析并执行模板
	tpl, err := template.New("common").Parse(cJsonTemplate)
	if err != nil {
		// 处理模板解析错误
		panic(err)
	}
	if err := tpl.Execute(&content, data); err != nil {
		// 处理模板执行错误
		panic(err)
	}

	file.AddBuilder(&content)
}

// 定义过滤器代码生成模板
const filterTemplate = `func {{.FilterName}}(c *gin.Context) {
	res := {{.ImportName}}.{{.FunctionName}}(c, &c.Request)
	if res.Code != 0 {
		cJSON(c, 200, Response{
			Code:    int(res.Code),
			Message: res.Message,
		})
		c.Abort()
	}
}
`

func (servlet *ServletGen) GenFilterCode(function *astinfo.Function, file *astinfo.GenedFile) string {
	file.GetImport(astinfo.SimplePackage("github.com/gin-gonic/gin", "gin"))
	pkg := function.GoSource.Pkg
	// 生成过滤器函数名
	filterName := "filter_" + pkg.Name + "_" + function.Name
	impt := file.GetImport(pkg)

	// 准备模板数据
	data := struct {
		FilterName   string
		ImportName   string
		FunctionName string
	}{}
	data.FilterName = filterName
	data.ImportName = impt.Name
	data.FunctionName = function.Name

	// 解析并执行模板
	var sb strings.Builder
	tpl, err := template.New("filter").Parse(filterTemplate)
	if err != nil {
		panic(err)
	}
	if err := tpl.Execute(&sb, data); err != nil {
		panic(err)
	}

	file.AddBuilder(&sb)

	// 处理URL注释逻辑
	if function.Comment.Url == "" || function.Comment.Url == "\"\"" {
		return filterName
	} else {
		filterInfo := &FilterInfo{
			FilterName:    filterName,
			FilterRawName: function.Name,
			Func:          function,
		}
		servlet.filterMap[function.Name] = filterInfo
		servlet.filters = append(servlet.filters, filterInfo)
		return ""
	}
}

// genRouterCode
func (servlet *ServletGen) GenRouterCode(method *astinfo.Method, file *astinfo.GenedFile) string {
	name := ""
	var sb strings.Builder
	file.AddBuilder(&sb)
	// method.generateServletPostCall(file, &sb)
	// var realParams string
	// var rawServlet = false
	//  有request请求，需要解析request，有些情况下，服务端不需要request；
	// 参数为context.Context, request *schema.Request
	type CodeParam struct {
		HttpMethod       string
		MethodName       string
		Url              string
		FilterName       string //自带最后一个逗号
		RequestConstruct string
		UrlParameterStr  string
		HasRequest       bool
		HasResponse      bool
		ResponseNilCode  string
		DataError        int
	}
	tm := &CodeParam{
		HttpMethod: method.Comment.Method,
		MethodName: method.Name,
		Url:        path.Join(method.Receiver.Comment.Url, method.Comment.Url),
		DataError:  servlet.DataError,
	}
	if len(method.Params) > 1 {
		paramIndex := 1
		requestParam := method.Params[paramIndex]
		if !astinfo.IsPointer(requestParam.Type) {
			fmt.Printf("only pointer is supported in %s of file %s \n", method.Name, method.GoSource.Path)
			os.Exit(0)
		}
		tm.HasRequest = true
		tm.RequestConstruct = requestParam.GenVariableCode(file, false)
	}
	if len(method.Results) > 1 {
		tm.HasResponse = true
		// tm.ResponseNilCode = method.Results[0].GenNilCode(file)
	}

	//获取可能存在的url中的参数
	methodUrl := strings.Trim(method.Comment.Url, "\"")
	if strings.Contains(methodUrl, ":") {
		names := strings.Split(methodUrl, "/")
		for _, name := range names {
			if strings.Contains(name, ":") {
				//此处最好从名字能获取到Field，然后在调用type的parse方法，返回其对应的值；
				tm.UrlParameterStr += fmt.Sprintf("request.%s=c.Param(\"%s\")\n", astinfo.Capitalize(name[1:]), name[1:])
			}
		}
	}
	userFilters := strings.Split(method.Comment.Filter, ",")
	for _, filter := range userFilters {
		filter = strings.Trim(filter, "\t ")
		if filter != "" {
			filterInfo := servlet.filterMap[filter]
			if filterInfo == nil {
				fmt.Printf("filter %s not found in file %s for %s \n", filter, method.GoSource.Path, method.Name)
			} else {
				tm.FilterName += filterInfo.FilterName + ","
			}
		}
	}
	for _, filter := range servlet.filters {
		if strings.Contains(methodUrl, filter.Func.Comment.Url) {
			tm.FilterName += filter.FilterName + ","
		}
	}
	tmplText := `engine.{{.HttpMethod}} ( "{{.Url}}", {{.FilterName}} func(c *gin.Context) {
		{{ if .HasRequest }}
		request := {{.RequestConstruct}}
		{{.UrlParameterStr}}	
		// 利用gin的自动绑定功能，将请求内容绑定到request对象上；兼容get,post等情况
		if err := c.ShouldBind(request); err != nil {
			cJSON(c, 200, Response{
				Code:    {{.DataError}},
				Message: "param error",
			})
			return
		}
		{{ end }}
		{{ if .HasResponse }}a,{{end}} err := receiver.{{.MethodName}}(c {{ if .HasRequest }},request{{ end }})
		{{.ResponseNilCode}}
		var code = 200
		errorCode,errMessage:=getErrorCode(err)
		cJSON(c, code, Response{
			{{ if .HasResponse }}Object:  a,{{ end }}
			Code:    errorCode,
			Message: errMessage,
		})
	})
		`

	tmpl, err := template.New("personInfo").Parse(tmplText)
	if err != nil {
		log.Fatalf("解析模板失败: %v", err)
	}
	err = tmpl.Execute(&sb, tm)
	if err != nil {
		log.Fatalf("执行模板失败: %v", err)
	}
	return name
}
