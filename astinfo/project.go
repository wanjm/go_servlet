package astinfo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	Path    string              // 项目所在的目录
	Mod     string              // 该项目的mode名字
	Package map[string]*Package //key是mod的全路径
}

func (project *Project) Parse() {
	//读取go.mod
	project.Mod = "gitlab.plaso.cn/bisshow"
	project.parseDir(project.Path)
}

func CreateProject(path string) Project {
	return Project{
		Path:    path,
		Package: make(map[string]*Package),
	}
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

// 根据dir全路径，返回mod全路径
func (project *Project) getModePath(pathStr string) string {
	pathLen := len(project.Path)
	if !strings.HasPrefix(pathStr, project.Path) {
		log.Fatalf("pack path %s is not in current Dir %s\n", pathStr, project.Path)
	}
	return filepath.Join(project.Mod, pathStr[pathLen:])
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

func (project *Project) GenerateCode() string {
	var sb strings.Builder
	//生成函数明
	sb.WriteString("func InitRoute(router *gin.Engine) {\n")
	//生成原始初始化对象，如数据库等；
	//生成servlet
	for _, pkg := range project.Package {
		sb.WriteString(pkg.GenerateCode())
	}
	sb.WriteString("}\n")
	return sb.String()
}
