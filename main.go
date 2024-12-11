package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/wanjm/go_servlet/astinfo"
)

func main() {
	var path string
	flag.StringVar(&path, "p", ".", "需要生成代码工程的根目录")
	init := flag.Bool("i", false, "初始化文件")
	h := flag.Bool("h", false, "显示帮助文件")
	flag.Parse()
	if *h {
		flag.Usage()
		return
	}
	path, err := filepath.Abs(path)
	if err != nil {
		log.Printf("open %s failed with %s", path, err.Error())
		return
	}
	os.Chdir(path)
	cfg := astinfo.Config{
		InitMain: *init,
	}
	cfg.Load()
	var project = astinfo.CreateProject(path, &cfg)
	project.Parse()
	project.GenerateCode()
}
