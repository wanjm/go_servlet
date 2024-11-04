package astinfo

import (
	"os"

	"github.com/BurntSushi/toml"
)

type SwaggerCfg struct {
	ProjectId     string
	ServletFolder string
	SchemaFolder  string
	Token         string
}
type Config struct {
	SwaggerCfg SwaggerCfg
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
