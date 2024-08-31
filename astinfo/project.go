package astinfo

import (
	"fmt"
	"os"
	"path/filepath"
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
	var pack = Package{Project: project}
	pack.Parse(pathStr)
	project.Package = append(project.Package, &pack)
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
