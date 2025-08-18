package astinfo

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"text/template"
)

type MainProject struct {
	currentProject Project
	Packages       map[string]*Package // 项目包含的包集合（key为包全路径）
	Cfg            *Config

	*InitManager
	InitFuncs4All    []string   // 启动服务器和启动test都是用的方法；
	InitFuncs4Server []string   // 启动服务器用的方法；
	Projects         []*Project // 项目包含的子项目集合（key为Project的module）
}

func (mp *MainProject) genGoMod() {
	_, err := os.Stat("go.mod")
	if os.IsNotExist(err) {
		var content = "module " + mp.Cfg.InitMain + "\n" + strings.Replace(runtime.Version(), "go", "go ", 1) + "\n"
		os.WriteFile("go.mod", []byte(content), 0660)
	}
}

// genMain
func (mp *MainProject) genMain() {
	var content strings.Builder
	content.WriteString("package main\n")
	//	import "gitlab.plaso.cn/message-center/gen"
	content.WriteString("import (\"flag\"\n\"" + mp.currentProject.Module + "/gen\")\n")
	content.WriteString(`
func main() {
	parseArgument();
	run()
}
func parseArgument() {
	flag.Parse()
}
func run() {
	wg:=gen.Run(gen.Config{
		Cors: true,
		Addr: ":8080",
		ServerName: "servlet", // this is the name of group tag in comments;
	})
	wg.Wait()
}
	`)
	os.WriteFile("main.go", []byte(content.String()), 0660)

}

// genBasic 生成basic.go
func (mp *MainProject) genBasic() {
	os.Mkdir("basic", 0750)
	os.WriteFile("basic/message.go", []byte(`package basic
type Error struct {
	Code    int    "json:\"code\""
	Message string "json:\"message\""
}

func (error *Error) Error() string {
	return error.Message
}
func New(code int, msg string) error {
	res := &Error{
		Code:    code,
		Message: msg,
	}
	return res
}
func (error *Error) GetErrorCode() int {
	return error.Code
}
	`), 0660)
}

func (mp *MainProject) genProjectCode() {
	err := os.Mkdir("gen", 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	file := createGenedFile("goservlet_project")
	file.GetImport(SimplePackage("github.com/gin-gonic/gin", "gin"))
	os.Chdir("gen")
	mp.genBasicCode(file)
	mp.genPrepare(file)
	file.save()
}
func (mp *MainProject) genPrepare(file *GenedFile) {

	mp.InitInitorator()
	mp.InitManager.Generate(file)
	mp.InitManager.GenterateTestCode(file)

	sm := CreateServerManager()
	sm.Prepare()
	sm.Generate(file)

	cm := NewRpcClientManager()
	cm.Prepare()
	cm.Generate(file)

	// 定义模板字符串
	const prepareTemplate = `
// gened by mp.genPrepare
func Prepare() {
	//from mp.InitFuncs4All
{{range .InitFuncs4All}}	{{.}}()
{{end}}}

func prepare() {
	Prepare()
	//from mp.InitFuncs4Server
{{range .InitFuncs4Server}}	{{.}}()
{{end}}}
// gened by mp.genPrepare
`

	// 创建并解析模板
	tpl, err := template.New("prepare").Parse(prepareTemplate)
	if err != nil {
		panic("Failed to parse prepare template: " + err.Error())
	}

	// 渲染模板到strings.Builder
	var content strings.Builder
	if err := tpl.Execute(&content, mp); err != nil {
		panic("Failed to execute prepare template: " + err.Error())
	}

	file.AddBuilder(&content)
}
func (mp *MainProject) genBasicCode(file *GenedFile) {
	file.GetImport(SimplePackage("github.com/gin-contrib/cors", "cors"))
	file.GetImport(SimplePackage("sync", "sync"))

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
	ServerName string
}
func getAddr[T any](a T)*T{
	return &a
}
type server struct {
	filters      gin.HandlersChain
	routerInitors []func(*gin.Engine)
}
var servers map[string]*server
	func Run(config ...Config) *sync.WaitGroup{
		prepare()
		var wg sync.WaitGroup
		for _, c := range config {
			wg.Add(1)
			go run(&wg, c)
		}
		return &wg
	}
	func run(wg *sync.WaitGroup, config Config){
		var	router  *gin.Engine = gin.New()
		router.ContextWithFallback = true
		if(config.Cors){
			config := cors.DefaultConfig()
			config.AllowAllOrigins = true
			config.AllowHeaders = append(config.AllowHeaders, "*")
			router.Use(cors.New(config))
		}
		register(config.ServerName, router)
		if config.CertFile != "" {
			router.RunTLS(config.Addr, config.CertFile, config.KeyFile)
		} else {
			router.Run(config.Addr)
		}
		wg.Done()
	}
		const TraceId = "TraceId"
	`)

	file.AddBuilder(&content)
}

type Server struct {
	Name             string
	callGen          CallableGen
	GeneratedFilters []string
	GenerateRouters  []string
	filters          []*Function
	routers          []*Struct
	// initRouteFuns []string           //initRoute 调用的init函数； 有package生成，生成路由代码时生成，一个package生成一个路由代码
	// urlFilters    map[string]*Filter //记录url过滤器函数,key是url, url是原始文件中的url，可能包含引号
	// initFuncs     []string           //initAll 调用的init函数；
}

// generate
func (sm *Server) Generate(file *GenedFile) {
	generator := sm.callGen
	generator.GenerateCommon(file)
	for _, function := range sm.filters {
		filterName := generator.GenFilterCode(function, file)
		if filterName == "" {
			continue
		}
		sm.GeneratedFilters = append(sm.GeneratedFilters, filterName)
	}
	for _, class := range sm.routers {
		//generate begin;
		sm.GenerateRouters = append(sm.GenerateRouters, sm.generateBegin(class, file))

		// generate servlets;
		for _, method := range class.MethodManager.Server {
			generator.GenRouterCode(method, file)
		}

		// generate end
		var end strings.Builder
		end.WriteString("}\n")
		file.AddBuilder(&end)
	}
}

// generateBegin
func (sm *Server) generateBegin(class *Struct, file *GenedFile) string {
	var name = strings.Join([]string{
		"init",
		class.Comment.GroupName,
		class.goSource.Pkg.Name,
		class.StructName,
		"router",
	}, "_")
	var declare strings.Builder
	var receiver = Variable{
		Type: NewPointerType(class),
		Name: "receiver",
		Wire: true,
	}
	declare.WriteString("func " + name + "(engine *gin.Engine) {\n")
	declare.WriteString(receiver.Name + ":=" + receiver.Generate(file))
	declare.WriteString("\n")
	file.AddBuilder(&declare)
	file.GetImport(SimplePackage("github.com/gin-gonic/gin", "gin"))
	return name
}

// 负责对配置的每个server进行初始化，管理其中的filter，servlet；并生成最终代码中的server代码。打通filter和servlet的注册环节
// 其生成代码分为连个部分；
// 1. 最终代码的server代码。完成代码的filter和路由的注册；
// 2. filter，和路由的工作代码
type ServerManager struct {
	servers   map[string]*Server
	generator map[string]CallableGen
}

func CreateServerManager() *ServerManager {
	sm := &ServerManager{
		servers:   map[string]*Server{},
		generator: map[string]CallableGen{},
	}
	return sm
}

// register
func (sm *ServerManager) register(callGen CallableGen) {
	name := callGen.GetName()
	sm.generator[name] = callGen
}

func (sm *ServerManager) Prepare() {
	for _, callGen := range callableGens {
		sm.register(callGen)
	}
	sm.splitServers()
}

// 扫描所有的程序，将服务按照group分为多个server；
func (sm *ServerManager) splitServers() {
	project := GlobalProject
	for _, pkg := range project.Packages {
		// 结构体会定义group和type，所以先扫描struct
		for _, router := range pkg.Structs {
			var server *Server
			var ok bool
			var groupName = router.Comment.GroupName
			if groupName == "" {
				continue
			}
			if server, ok = sm.servers[groupName]; !ok {
				gen := sm.generator[router.Comment.serverType]
				if gen == nil {
					fmt.Printf("failed to found generator %s\n", router.Comment.serverType)
					continue
				}
				server = &Server{
					Name:    groupName,
					callGen: gen,
				}
				sm.servers[groupName] = server
			}
			server.routers = append(server.routers, router)
		}
	}
	for _, pkg := range project.Packages {
		for _, filter := range pkg.Filter {
			var server *Server
			var ok bool
			var groupName = filter.Comment.groupName
			if server, ok = sm.servers[groupName]; ok {
				server.filters = append(server.filters, filter)
			} else {
				fmt.Printf("failed to found server %s\n", groupName)
			}
		}
	}
}

// Generate
func (sm *ServerManager) Generate(file *GenedFile) {
	// if len(sm.servers) == 0 {
	// 	return
	// }
	for _, server := range sm.servers {
		//一个server一个文件；
		file1 := createGenedFile(server.Name)
		server.Generate(file1)
		file1.save()
	}
	tmplText :=
		`
func initServer(){
	servers = make(map[string]*server)
	{{range .}}
	servers["{{.Name}}"] = &server{
		filters: gin.HandlersChain{	{{.FilterNames}} },
		routerInitors: []func(*gin.Engine){ {{.RouterNames}} },
	}
	{{end}}
}

	func register(name string, router *gin.Engine ){
		server := servers[name]
		if server.filters != nil {
			router.Use(server.filters...)
		}
		for _, routerInitor := range server.routerInitors {
			routerInitor(router)
		}
	}
`
	tmpl, err := template.New("personInfo").Parse(tmplText)
	if err != nil {
		log.Fatalf("解析模板失败: %v", err)
	}
	var sb strings.Builder
	type ServerInfo struct {
		Name        string
		FilterNames string
		RouterNames string
	}
	var s []*ServerInfo
	for _, server := range sm.servers {
		server := &ServerInfo{
			Name:        server.Name,
			FilterNames: strings.Join(server.GeneratedFilters, ",\n"),
			RouterNames: strings.Join(server.GenerateRouters, ",\n"),
		}
		s = append(s, server)
	}

	err = tmpl.Execute(&sb, s)
	if err != nil {
		log.Fatalf("执行模板失败: %v", err)
	}
	GlobalProject.InitFuncs4Server = append(GlobalProject.InitFuncs4Server, "initServer")
	file.AddBuilder(&sb)
}

// GetPackage retrieves a package by module path without creation
func (mp *MainProject) GetPackage(module string) *Package {
	return mp.Packages[module]
}

// FindPackage finds or creates a package with automatic module path resolution
func (mp *MainProject) FindPackage(module string) *Package {
	if pkg := mp.GetPackage(module); pkg != nil {
		return pkg
	}
	if module == mp.currentProject.Module+"/gen" {
		newPkg := NewPackage(module, true, path.Join(mp.currentProject.Path, "gen"))
		newPkg.finshedParse = true
		newPkg.Name = "gen"
		return newPkg
	}

	for _, p := range mp.Projects {
		// 根据module寻找package
		if p.Module == "" {
			panic(fmt.Sprintf("project module is empty %s\n", p.Path))
		}
		if strings.HasPrefix(module, p.Module) {
			newPkg := NewPackage(module, p.Simple, path.Join(p.Path, module[len(p.Module):]))
			mp.Packages[module] = newPkg
			newPkg.SimpleParse()
			return newPkg
		}
	}
	newPkg := NewSysPackage(module)
	newPkg.SimpleParse()
	mp.Packages[module] = newPkg
	//此处识别为系统Package
	return newPkg
}

// GenerateCode 生成项目的代码
func (mp *MainProject) GenerateCode() error {
	// 遍历所有包
	for _, pkg := range mp.Packages {
		_ = pkg
		// 生成包的代码
		// if err := pkg.GenerateCode(); err != nil {
		// 	return fmt.Errorf("error generating code for package %s: %w", pkg.Name, err)
		// }
	}
	if mp.Cfg.InitMain != "" {
		mp.genMain()
		mp.genBasic()
	}
	mp.genProjectCode()

	//NewSwagger(mp).GenerateCode(&mp.Cfg.SwaggerCfg)
	return nil
}
func escapeModulePath(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune('!')
			result.WriteRune(r - 'A' + 'a') // 转为小写
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Parse 解析项目的代码
func (mp *MainProject) Parse() error {
	if mp.Cfg.InitMain != "" { // 检查是否非空字符串
		mp.genGoMod()
		// 设置项目模块名称
		mp.currentProject.Module = mp.Cfg.InitMain
	}
	p := &mp.currentProject
	if err := p.ParseModule(); err != nil {
		return err
	}
	cfg := mp.Cfg
	traceKeyMod := cfg.Generation.TraceKeyMod
	if !strings.Contains(traceKeyMod, ".") {
		cfg.Generation.TraceKeyMod = p.Module + "/" + traceKeyMod
	}
	responseMod := cfg.Generation.ResponseMod
	if !strings.Contains(responseMod, ".") {
		cfg.Generation.ResponseMod = p.Module + "/" + responseMod
	}
	mp.Projects = append(mp.Projects, p)
	goPath := os.Getenv("GOPATH")
	for _, mod := range p.Require {
		// if mod.Indirect {
		// 	continue
		// }

		p := Project{
			Path:   path.Join(goPath, "pkg/mod", escapeModulePath(mod.Mod.Path)+"@"+mod.Mod.Version),
			Simple: true,
		}
		p.ParseModule()
		if p.Module == "" {
			p.Module = mod.Mod.Path
		}
		mp.Projects = append(mp.Projects, &p)
	}
	sort.Slice(mp.Projects, func(i, j int) bool {
		return mp.Projects[i].Module > mp.Projects[j].Module
	})
	return p.ParseCode()
}

var GlobalProject *MainProject

func CreateProject(path string, cfg *Config) *MainProject {
	GlobalProject = &MainProject{
		Cfg:      cfg,
		Packages: make(map[string]*Package),
		// initiatorMap: make(map[*Struct]*Initiators),
		// servers:      make(map[string]*server),
		// creators: make(map[*Struct]*Initiator),
	}
	GlobalProject.currentProject.Path = path
	// 由于Package中有指向Project的指针，所以RawPackage指向了此处的project，如果返回对象，则出现了两个Project，一个是返回的Project，一个是RawPackage中的Project；
	// 返回*Project才能保证这是一个Project对象；
	// project.initRawPackage()
	// project.rawPkg = project.getPackage("", true)
	return GlobalProject
}
