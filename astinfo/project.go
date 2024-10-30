package astinfo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	Path       string              // 项目所在的目录
	Mod        string              // 该项目的mode名字
	Package    map[string]*Package //key是mod的全路径
	urlFilters []*Function         //记录url过滤器函数
	// creators map[*Struct]*Initiator
	initFuncs        []string                //initAll 调用的init函数；
	initVariableFuns []string                //initVriable 调用的init函数； 由package生成代码时，处理initiator函数生成；
	initRouteFuns    []string                //initRoute 调用的init函数； 有package生成，生成路由代码时生成，一个package生成一个路由代码
	initRpcField     []*Field                //initRpcClient 调用的init函数；主要是给每个initClient调用
	initiatorMap     map[*Struct]*Initiators //便于注入时根据类型存照
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
	project.initRawPackage()
	return project
}
func (project *Project) initRawPackage() {
	rawPkg := project.getPackage(GolangRawType, true) //创建原始类型
	rawPkg.getStruct("string", true)
	rawPkg.getStruct("bool", true)
	rawPkg.getStruct("byte", true)
	rawPkg.getStruct("rune", true)
	rawPkg.getStruct("int", true)
	rawPkg.getStruct("int8", true)
	rawPkg.getStruct("int16", true)
	rawPkg.getStruct("int32", true)
	rawPkg.getStruct("int64", true)
	rawPkg.getStruct("uint", true)
	rawPkg.getStruct("uint8", true)
	rawPkg.getStruct("uint16", true)
	rawPkg.getStruct("uint32", true)
	rawPkg.getStruct("uint64", true)
	rawPkg.getStruct("float32", true)
	rawPkg.getStruct("float64", true)
	rawPkg.getStruct("array", true)
	rawPkg.getStruct("map", true)
}

func (project *Project) addUrlFilter(function *Function) {
	project.urlFilters = append(project.urlFilters, function)
}
func (project *Project) getPackage(modPath string, create bool) *Package {
	pkg := project.Package[modPath]
	if pkg == nil && create {
		// fmt.Printf("create package %s\n", modPath)
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
	// fmt.Printf("parse %s\n", pathStr)
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

// 根据扫描情况生成filter函数；
func (project *Project) generateUrlFilter(file *GenedFile) {
	if len(project.urlFilters) == 0 {
		return
	}
	var content strings.Builder
	var result0 = project.urlFilters[0].Results[0]
	file.getImport("context", "context")
	file.getImport("net/http", "http")
	file.getImport("strings", "strings")
	pkg := result0.pkg
	file.getImport(pkg.modPath, pkg.modName)

	content.WriteString(`
	type UrlFilter struct {
		path     string
		function func(c context.Context, Request *http.Request) (error basic.Error)
	}
	func registerFilter(router *gin.Engine) {
	`)

	content.WriteString("var urlFilters =[]*UrlFilter{\n")
	for _, filter := range project.urlFilters {
		impt := file.getImport(filter.pkg.modPath, filter.pkg.modName)
		content.WriteString(fmt.Sprintf("{path:%s, function:%s.%s},\n", filter.Url, impt.Name, filter.Name))
	}
	content.WriteString(`
		}
		router.Use(func(ctx *gin.Context) {
			path := ctx.Request.URL.Path
			for _, filter := range urlFilters {
				if strings.Contains(path, filter.path) {
					error := filter.function(ctx, ctx.Request)
					if error.Code != 0 {
						ctx.JSON(400, error)
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
	file.addBuilder(&content)
	project.initFuncs = append(project.initFuncs, "registerFilter(router)")
}

// file:
// 	package gen
//	import

// 	Response
// 	InitAll

// UrlFilter
// registerFilter
// initVariable
// initRoute

// RpcClient
// initRpcClient
func (project *Project) GenerateCode() {
	os.Chdir(project.Path)
	err := os.Mkdir("gen", 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	file := createGenedFile("project")
	file.getImport("github.com/gin-gonic/gin", "gin")
	os.Chdir("gen")
	// project.generateInit(&content)

	// 根据情况生成filter函数；

	//生成函数明

	//生成原始初始化对象，如数据库等；
	//分步完成的原因是保证所有的变量都提前生成，后续注入可以找到变量
	for _, pkg := range project.Package {
		var name string
		name = project.getRelativeModePath(pkg.modPath)
		name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
		pkg.file = createGenedFile(name)
		pkg.generateInitorCode()
	}

	for _, pkg := range project.Package {
		pkg.GenerateRouteCode()
		pkg.GenerateRpcClientCode()
		pkg.file.save()
	}

	project.genInitVariable(file)
	project.generateUrlFilter(file)
	project.genRpcClientVariable(file)
	project.genInitRoute(file)
	project.genInitAll(file)
	file.save()
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

func (funcManager *Project) genRpcClientVariable(file *GenedFile) {
	if len(funcManager.initRpcField) == 0 {
		return
	}

	file.getImport("bytes", "bytes")
	file.getImport("encoding/json", "json")
	file.getImport("fmt", "fmt")
	file.getImport("net/http", "http")

	var content strings.Builder
	content.WriteString("func initRpcClient() {\n")
	for _, field := range funcManager.initRpcField {
		impt := file.getImport(field.pkg.modPath, field.pkg.modName)
		cfg := field.pkg.getInterface(field.typeName, false).config
		content.WriteString(fmt.Sprintf("%s.%s = &%sStruct{client:RpcClient{Prefix:%s+\":\"+%s}}\n", impt.Name, field.name, field.typeName, cfg.Port, cfg.Host))
	}
	content.WriteString("}\n")
	content.WriteString(`
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
	`)
	file.addBuilder(&content)
	funcManager.addInitFuncs("initRpcClient()")
}

func (project *Project) addInitVariable(variableName string) {
	project.initVariableFuns = append(project.initVariableFuns, variableName+"()")
}

func (project *Project) addInitRoute(routerName string) {
	project.initRouteFuns = append(project.initRouteFuns, routerName+"(router)")
}

func (project *Project) addInitFuncs(rpcClientName string) {
	project.initFuncs = append(project.initFuncs, rpcClientName)
}

// initRpcClientFuns
func (project *Project) addInitRpcClientFuns(rpcField *Field) {
	project.initRpcField = append(project.initRpcField, rpcField)
}

func (project *Project) genInitRoute(file *GenedFile) {
	if len(project.initRouteFuns) == 0 {
		return
	}
	var content strings.Builder
	content.WriteString("func initRoute(router *gin.Engine) {\n")
	for _, fun := range project.initRouteFuns {
		content.WriteString(fun + "\n")
	}
	content.WriteString("}\n")
	file.addBuilder(&content)
	project.addInitFuncs("initRoute(router)")
}

func (Project *Project) genInitVariable(file *GenedFile) {
	if len(Project.initVariableFuns) == 0 {
		return
	}
	var content strings.Builder
	content.WriteString("func initVariable() {\n")
	for _, fun := range Project.initVariableFuns {
		content.WriteString(fun + "\n")
	}
	content.WriteString("}\n")
	file.addBuilder(&content)
	Project.addInitFuncs("initVariable()")
}

func (Project *Project) genInitAll(file *GenedFile) {
	var content strings.Builder
	content.WriteString(`
	type Response struct {
		Code    int         "json:\"code\""
		Message string      "json:\"message,omitempty\""
		Object  interface{} "json:\"obj,omitempty\""
	}
	func InitAll(router *gin.Engine){
	`)
	for _, fun := range Project.initFuncs {
		content.WriteString(fun + "\n")
	}
	content.WriteString("}\n")
	file.addBuilder(&content)
}
