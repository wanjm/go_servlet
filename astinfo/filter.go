package astinfo

import "strings"

type Filter struct {
	function *Function
	server   *server
	url      string //filter的url,不包含引号
	genName  string
}

func newFilter(url string, function *Function) *Filter {
	return &Filter{
		function: function,
		url:      strings.Trim(url, "\""),
	}
}
func (filter *Filter) genFilterCode(file *GenedFile) {
	file.getImport("github.com/gin-gonic/gin", "gin")
	pkg := filter.function.pkg
	//生成这个函数，pkg.file已经生成了，所以可以直接使用
	filter.genName = "filter_" + pkg.file.name + "_" + filter.function.Name
	impt := file.getImport(pkg.modPath, pkg.modName)
	var sb = strings.Builder{}
	sb.WriteString("//filter_${pkg.file.name}_${filter.function.Name}\n")
	sb.WriteString("func ")
	sb.WriteString(filter.genName)
	sb.WriteString("(c *gin.Context) {\nres:=")
	sb.WriteString(impt.Name + "." + filter.function.Name)
	sb.WriteString(`(c,&c.Request)
	if(res.Code!=0){
			c.JSON(200, 
			Response{
				Code:int(res.Code),
				Message: res.Message,
			})
			c.Abort()
		}
	}
	`)
	file.addBuilder(&sb)
}
