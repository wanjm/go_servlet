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
type Generation struct {
	TraceKey    string
	TraceKeyMod string
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
