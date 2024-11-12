package astinfo

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/spec"
)

type server struct {
	name          string
	initRouteFuns []string             //initRoute 调用的init函数； 有package生成，生成路由代码时生成，一个package生成一个路由代码
	urlFilters    map[string]*Function //记录url过滤器函数
	initFuncs     []string             //initAll 调用的init函数；
}
type Project struct {
	cfg     *Config
	Path    string              // 项目所在的目录
	Mod     string              // 该项目的mode名字
	Package map[string]*Package //key是mod的全路径
	servers map[string]*server  //key是server的名字，default，prpcserver
	// creators map[*Struct]*Initiator
	initFuncs        []string                //initAll 调用的init函数；
	initiatorMap     map[*Struct]*Initiators //便于注入时根据类型存照
	initVariableFuns []string                //initVriable 调用的init函数； 由package生成代码时，处理initiator函数生成；
	initRpcField     []*Field                //initRpcClient 调用的init函数；主要是给每个initClient调用
	initMain         bool
	swag             *spec.Swagger
}

func (project *Project) Parse() {
	//读取go.mod
	modFile, err := os.Open("go.mod")
	if err != nil {
		log.Panicf("failed to open go.mod with error %s\n", err.Error())
		return
	}
	defer modFile.Close()
	scanner := bufio.NewScanner(modFile)
	// 读取第一行
	if scanner.Scan() {
		firstLine := scanner.Text()
		project.Mod = strings.Trim(strings.Split(firstLine, " ")[1], " \t")
	} else {
		log.Panicf("failed to read go.mod, please run 'go mod init' first\n")
		return
	}
	project.parseDir(project.Path)
}

func CreateProject(path string, cfg *Config) *Project {
	project := Project{
		Path:         path,
		Package:      make(map[string]*Package),
		initiatorMap: make(map[*Struct]*Initiators),
		cfg:          cfg,
		servers:      make(map[string]*server),
		// creators: make(map[*Struct]*Initiator),
	}
	// 由于Package中有指向Project的指针，所以RawPackage指向了此处的project，如果返回对象，则出现了两个Project，一个是返回的Project，一个是RawPackage中的Project；
	// 返回*Project才能保证这是一个Project对象；
	project.initRawPackage()
	return &project
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

func getRawTypeString(typeName string) string {
	switch typeName {
	case "string":
		return "string"
	case "array":
		return "array"
	case "map":
		return "object"
	case "bool":
		return "bool"
	case "float32", "float64":
		return "number"
	default:
		return "integer"
	}
}

func (project *Project) addServer(name string) {
	if _, ok := project.servers[name]; !ok {
		project.servers[name] = &server{name: name}
		return
	}
}
func (project *Project) addUrlFilter(function *Function, serverName string) {
	var s *server
	if s = project.servers[serverName]; s == nil {
		s = &server{name: serverName, urlFilters: make(map[string]*Function)}
		project.servers[serverName] = s
	}
	if filter, ok := s.urlFilters[function.comment.Url]; ok {
		log.Fatalf("url %s has been defined in %s\n", function.comment.Url, filter.pkg.modPath)
	} else {
		s.urlFilters[function.comment.Url] = function
	}
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

func (server *server) addInitFuncs(rpcClientName string) {
	server.initFuncs = append(server.initFuncs, rpcClientName)
}

func (server *server) genInitRoute(file *GenedFile) {
	if len(server.initRouteFuns) == 0 {
		return
	}
	var content strings.Builder
	content.WriteString("func initRoute(router *gin.Engine) {\n")
	for _, fun := range server.initRouteFuns {
		content.WriteString(fun + "\n")
	}
	content.WriteString("}\n")
	file.addBuilder(&content)
	server.addInitFuncs("initRoute(router)")
}

// 根据扫描情况生成filter函数；
func (project *Project) generateUrlFilter(file *GenedFile) {
	var content strings.Builder

	file.getImport("context", "context")
	file.getImport("net/http", "http")
	file.getImport("strings", "strings")
	content.WriteString(`
	type UrlFilter struct {
		path     string
		//此处的basic.Error在代码生成时是写死的，还不够灵活，且宿主工程包中需要定义一个filter，否则代码会报告basic找不到
		function func(c context.Context, Request **http.Request) (error basic.Error)
	}
	func registerFilter(router *gin.Engine, urlFilters []*UrlFilter) {
		router.Use(func(ctx *gin.Context) {
			path := ctx.Request.URL.Path
			for _, filter := range urlFilters {
				if strings.Contains(path, filter.path) {
					error := filter.function(ctx, &ctx.Request)
					if error.Code != 0 {
						ctx.JSON(400, error)
						ctx.Abort()
						return
					}
				}
			}
			ctx.Next()
		})
	}
	`)
	file.addBuilder(&content)
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
	project.genInitMain()
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

	for name, pkg := range project.Package {
		_ = name
		// fmt.Printf("deal package %s\n", name)
		pkg.GenerateRouteCode()
		pkg.GenerateRpcClientCode()
		pkg.file.save()
	}
	project.genBasicCode(file)
	project.genInitVariable(file)
	project.genRpcClientVariable(file)
	// project.genInitRoute(file)
	project.genPrepare(file)
	file.save()
	NewSwagger(project).GenerateCode(&project.cfg.SwaggerCfg)
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

func (project *Project) addInitRoute(routerName string, serverName string) {
	if s, ok := project.servers[serverName]; ok {
		s.initRouteFuns = append(s.initRouteFuns, routerName)
	} else {
		project.servers[serverName] = &server{name: serverName, initRouteFuns: []string{routerName}}
	}
}

func (project *Project) addInitFuncs(rpcClientName string) {
	project.initFuncs = append(project.initFuncs, rpcClientName)
}

// initRpcClientFuns
func (project *Project) addInitRpcClientFuns(rpcField *Field) {
	project.initRpcField = append(project.initRpcField, rpcField)
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

func (Project *Project) genBasicCode(file *GenedFile) {
	file.getImport("github.com/gin-contrib/cors", "cors")
	var content strings.Builder
	content.WriteString(`
	type Response struct {
		Code    int         "json:\"code\""
		Message string      "json:\"message,omitempty\""
		Object  interface{} "json:\"obj,omitempty\""
	}

type Config struct {
	CertFile string
	KeyFile string
	Cors bool
	Addr string
}

type server struct {
	filters      []*UrlFilter
	routerInitors []func(*gin.Engine)
}
var servers map[string]*server
	func Run(config Config, serverName string){
		var	router  *gin.Engine = gin.Default()
		if(config.Cors){
			config := cors.DefaultConfig()
			config.AllowAllOrigins = true
			config.AllowHeaders = append(config.AllowHeaders, "*")
			router.Use(cors.New(config))
		}
			//如果不存在，则启动就失败，不需要检查
			server := servers[serverName]
	registerFilter(router, server.filters)
	for _, routerInitor := range server.routerInitors {
		routerInitor(router)
	}
		if config.CertFile != "" {
			router.RunTLS(config.Addr, config.CertFile, config.KeyFile)
		} else {
			router.Run(config.Addr)
		}
	}
	`)
	file.addBuilder(&content)
}
func (Project *Project) genPrepare(file *GenedFile) {
	var content strings.Builder
	// file.getImport("sync/atomic", "atomic")
	content.WriteString(`
	func Prepare() {
	`)
	for _, fun := range Project.initFuncs {
		content.WriteString(fun + "\n")
	}
	content.WriteString("servers = make(map[string]*server)\n")
	// servers[""] = &server{
	// 	filters: []*UrlFilter{
	// 		{path: "/nc/", function: filter.NcFilter},
	// 	},
	// 	routerInitors: []func(*gin.Engine){},
	// }
	var oneResult *Field

	for _, server := range Project.servers {
		content.WriteString(fmt.Sprintf("servers[\"%s\"] = &server{\n", server.name))
		content.WriteString("filters: []*UrlFilter{\n")
		for _, filter := range server.urlFilters {
			impt := file.getImport(filter.pkg.modPath, filter.pkg.modName)
			oneResult = filter.Results[0]
			content.WriteString(fmt.Sprintf("{path:%s, function:%s.%s},\n", filter.comment.Url, impt.Name, filter.Name))
		}
		content.WriteString("},\n")
		content.WriteString("routerInitors: []func(*gin.Engine){\n")
		// server.initRouteFuns
		for _, fun := range server.initRouteFuns {
			content.WriteString(fmt.Sprintf("%s,\n", fun))
		}
		content.WriteString("},\n")
		content.WriteString("}\n")
	}
	if oneResult != nil {
		// 动态方式添加 basic.Error;
		pkg := oneResult.pkg
		file.getImport(pkg.modPath, pkg.modName)
	}
	Project.generateUrlFilter(file)
	content.WriteString("}\n")
	file.addBuilder(&content)
}

func (project *Project) genInitMain() {
	//如果是空目录，或者init为true；则生成main.go 和basic.go的Error类；
	if !project.initMain {
		return
	}
	var content strings.Builder
	content.WriteString("package main\n")
	//	import "gitlab.plaso.cn/message-center/gen"
	content.WriteString("import (\"" + project.Mod + "/gen\")\n")
	content.WriteString(`
func main() {
	gen.Run(gen.Config{
		Cors: true,
		Addr: ":8080",
	},"servlet")
}
	`)
	os.WriteFile("main.go", []byte(content.String()), 0660)
	os.Mkdir("basic", 0750)
	os.WriteFile("basic/message.go", []byte(`package basic
type Error struct {
	Code    int    "json:\"code\""
	Message string "json:\"message\""
}

func (error *Error) Error() string {
	return error.Message
}
	`), 0660)
}
