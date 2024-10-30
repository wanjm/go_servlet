package main

import (
	"flag"
	"log"
	"path/filepath"

	"gitlab.plaso.cn/webgen/astinfo"
)

func main() {
	var path string
	flag.StringVar(&path, "path", ".", "需要生成代码工程的根目录")
	flag.Parse()
	path, err := filepath.Abs(path)
	if err != nil {
		log.Printf("open %s failed with %s", path, err.Error())
		return
	}
	var project = astinfo.CreateProject(path)
	project.Parse()
	project.GenerateCode()
}
