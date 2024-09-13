package astinfo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const TagPrefix = "@plaso"
const GolangRawType = "rawType"

type Project struct {
	Path         string              // 项目所在的目录
	Mod          string              // 该项目的mode名字
	Package      map[string]*Package //key是mod的全路径
	initiatorMap map[*Struct]*Initiators
	urlFilters   []*Function //记录url过滤器
	// creators map[*Struct]*Initiator
}

func (project *Project) Parse() {
	//读取go.mod
	project.Mod = "gitlab.plaso.cn/bisshow"
	project.parseDir(project.Path)
}

func CreateProject(path string) Project {
	project := Project{
		Path:         path,
		Package:      make(map[string]*Package),
		initiatorMap: make(map[*Struct]*Initiators),
		// creators: make(map[*Struct]*Initiator),
	}
	project.getPackage(GolangRawType, true) //创建原始类型
	return project
}

func (project *Project) addUrlFilter(function *Function) {
	project.urlFilters = append(project.urlFilters, function)
}
func (project *Project) getPackage(modPath string, create bool) *Package {
	pkg := project.Package[modPath]
	if pkg == nil && create {
		fmt.Printf("create package %s\n", modPath)
		pkg = CreatePackage(project, modPath)
		project.Package[modPath] = pkg
	}
	return pkg
}

func (project *Project) getRelativeModePath(fullModPath string) (name string) {
	projectModPathLen := len(project.Mod)
	if len(fullModPath) > projectModPathLen {
		name = fullModPath[projectModPathLen+1:]
	} else {
		name = "root"
	}
	return
}

// 根据dir全路径，返回mod全路径
func (project *Project) getModePath(pathStr string) string {
	pathLen := len(project.Path)

	if strings.HasPrefix(pathStr, project.Path) {
		pathStr = pathStr[pathLen:]
	}
	return filepath.Join(project.Mod, pathStr)
}

func (project *Project) parseDir(pathStr string) {
	fmt.Printf("parse %s\n", pathStr)
	pkg := project.getPackage(project.getModePath(pathStr), true)
	pkg.Parse(pathStr)
	list, err := os.ReadDir(pathStr)
	if err != nil {
		fmt.Printf("read %s failed skip parse\n", pathStr)
		return
	}
	for _, d := range list {
		// 后续添加配置，跳过扫描路径
		if d.IsDir() && d.Name() != "gen" && !strings.HasPrefix(d.Name(), ".") {
			project.parseDir(filepath.Join(pathStr, d.Name()))
		}
	}
}
func (project *Project) generateInit(sb *strings.Builder) {

}

// 根据扫描情况生成filter函数；
func (project *Project) generateUrlFilter(content *strings.Builder, file *GenedFile) bool {
	if len(project.urlFilters) == 0 {
		return false
	}
	var result0 = project.urlFilters[0].Results[0]
	file.getImport("context", "context")
	file.getImport("net/http", "http")
	file.getImport("strings", "strings")
	file.getImport(result0.class.Package.modPath, result0.class.Package.modName)

	content.WriteString(`
	type UrlFilter struct {
		path     string
		function func(c context.Context, Request *http.Request) (error basic.Error)
	}
	func registerFilter(router *gin.Engine) {
	`)

	content.WriteString("var urlFilters =[]*UrlFilter{\n")
	for _, filter := range project.urlFilters {
		content.WriteString(fmt.Sprintf("&{path:\"%s\", function:%s},\n", filter.Url, filter.Name))
	}
	content.WriteString(`
		}
		router.Use(func(ctx *gin.Context) {
			path := ctx.Request.URL.Path
			for _, filter := range urlFilters {
				if strings.Contains(path, filter.path) {
					error := filter.function(ctx, ctx.Request)
					if error.Code != 0 {
						ctx.JSON(400, gin.H{"error": error.Message})
						ctx.Abort()
						return
					}
					break
				}
			}
			ctx.Next()
		})
	}
	`)
	return true
}
func (project *Project) GenerateCode() string {
	os.Chdir(project.Path)
	err := os.Mkdir("gen", 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	file := createGenedFile()
	file.getImport("github.com/gin-gonic/gin", "gin")
	os.Chdir("gen")
	var content strings.Builder
	// project.generateInit(&content)

	//生成函数明
	content.WriteString("package gen\n")
	content.WriteString(file.genImport())
	// 根据情况生成filter函数；
	callRegister := ""
	if project.generateUrlFilter(&content, file) {
		callRegister = "registerFilter(router)\n"
	}
	content.WriteString(`
	type Response struct {
		Code    int         "json:\"code\""
		Message string      "json:\"message,omitempty\""
		Object  interface{} "json:\"obj\""
	}
	func InitAll(router *gin.Engine){
		initVariable()
	`)
	content.WriteString(callRegister)
	content.WriteString(`
		initRoute(router)
	}
	`)
	var routeContent strings.Builder
	var variableContent strings.Builder
	routeContent.WriteString("func initRoute(router *gin.Engine) {\n")
	variableContent.WriteString("func initVariable() {\n")
	//生成原始初始化对象，如数据库等；
	//生成servlet
	for _, pkg := range project.Package {
		pkg.file = createGenedFile()
		pkg.generateInitorCode()
	}
	for _, pkg := range project.Package {
		variableName, routerName := pkg.GenerateCode()
		if len(variableName) > 0 {
			variableContent.WriteString(variableName + "()\n")
		}
		if len(routerName) > 0 {
			routeContent.WriteString(routerName + "(router)\n")
		}
	}
	variableContent.WriteString("}\n")
	routeContent.WriteString("}\n")
	content.WriteString(variableContent.String())
	content.WriteString(routeContent.String())
	os.WriteFile("project.go", []byte(content.String()), 0660)
	return ""
}

func (funcManager *Project) addInitiatorVaiable(initiator *Variable) {
	// 后续添加排序功能
	// funcManager.initiator = append(funcManager.initiator, initiator)
	var inits *Initiators
	var ok bool
	if inits, ok = funcManager.initiatorMap[initiator.class]; !ok {
		inits = createInitiators()
		funcManager.initiatorMap[initiator.class] = inits
	}
	inits.addInitiator(initiator)

}

func (funcManager *Project) getVariable(class *Struct, varName string) string {
	inits := funcManager.initiatorMap[class]
	if inits == nil {
		return ""
	}
	return inits.getVariableName(varName)
}
