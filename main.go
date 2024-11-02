package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"gitlab.plaso.cn/webgen/astinfo"
)

func main() {
	var path string
	flag.StringVar(&path, "p", "/Users/wanjm/myfile/git/yxt_server/message-center", "需要生成代码工程的根目录")
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
	var project = astinfo.CreateProject(path, *init)
	project.Parse()
	project.GenerateCode()
}
