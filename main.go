package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wanjm/gos/astinfo"
	"github.com/wanjm/gos/astinfo/callable_gen"
	rpcgen "github.com/wanjm/gos/astinfo/rpc_gen"
)

func main() {
	var path string
	flag.StringVar(&path, "p", ".", "需要生成代码工程的根目录")
	var modName string
	flag.StringVar(&modName, "i", "", "指定模块名称")
	h := flag.Bool("h", false, "显示帮助文件")
	v := flag.Bool("v", false, "显示版本信息") // 添加-v参数
	flag.Parse()

	if *v { // 检查是否指定了-v参数
		fmt.Println("gos version 0.2.1") // 打印版本号
		return                           // 退出程序
	}

	if *h {
		flag.Usage()
		return
	}
	path, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("open %s failed with %s", path, err.Error())
		return
	}
	os.Chdir(path)
	cfg := astinfo.Config{
		InitMain: modName, // 直接赋值模块名称
	}
	cfg.Load()
	astinfo.RegisterCallableGen(
		callable_gen.NewServletGen(4, 1),
		&callable_gen.PrpcGen{},
		&callable_gen.ResutfulGen{},
		&callable_gen.RawGen{},
	)
	astinfo.RegisterClientGen(&rpcgen.PrpcGen{})
	var project = astinfo.CreateProject(path, &cfg)

	// 移除原来的判断，因为现在InitMain直接存储模块名称
	// if len(modName) > 0 {
	// 	project.CurrentProject().Module = modName
	// }

	err = project.Parse()
	if err != nil {
		fmt.Printf("parse project failed with %s", err.Error())
		return
	}
	project.GenerateCode()
}
