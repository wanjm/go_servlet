package astinfo

import (
	"os"

	"github.com/BurntSushi/toml"
)

type SwaggerCfg struct {
	ProjectId     string // 项目id
	ServletFolder string // 生成的servlet文件夹
	SchemaFolder  string // 生成的schema文件夹
	UrlPrefix     string // url前缀, 正式环境和本地的路径不一样
	Token         string
}

// 产生代码相关配置
type Generation struct {
	TraceKey    string // 用于定义traceKy的结构体名字；用于context中记录traceId
	TraceKeyMod string // 用于定义traceKy的结构体所在的包名；
}
type Config struct {
	SwaggerCfg SwaggerCfg
	Generation Generation
	InitMain   bool
}

func (config *Config) Load() {
	buf, err := os.ReadFile("project.public.toml")
	if err == nil {
		_, err = toml.Decode(string(buf), config)
		if err != nil {
			panic(err)
		}
	}
	buf, err = os.ReadFile("project.private.toml")
	if err == nil {
		_, err = toml.Decode(string(buf), config)
		if err != nil {
			panic(err)
		}
	}
}
