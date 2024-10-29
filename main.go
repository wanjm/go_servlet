package main

import (
	"log"
	"path/filepath"

	"gitlab.plaso.cn/webgen/astinfo"
)

func main() {
	path, err := filepath.Abs("../server")
	if err != nil {
		log.Printf("open %s failed with %s", path, err.Error())
		return
	}
	var project = astinfo.CreateProject(path)
	project.Parse()
	project.GenerateCode()
}
