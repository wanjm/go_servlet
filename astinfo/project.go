package astinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	Path    string //xian
	Mod     string
	Package []*Package
}

func (project *Project) Parse() {
	//读取go.mod
	project.parseDir(project.Path)
}
func (project *Project) parseDir(pathStr string) {
	var pack = CreatePackage(project)
	pack.Parse(pathStr)
	project.Package = append(project.Package, pack)
	list, err := os.ReadDir(pathStr)
	if err != nil {
		fmt.Printf("read %s failed skip parse\n", pathStr)
		return
	}
	for _, d := range list {
		if d.IsDir() {
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
