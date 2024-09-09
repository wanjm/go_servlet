package astinfo

import (
	"log"
	"os"
	"strings"
)

// type Initiator struct {
// 	name     string    //用于被初始化变量的名字；当相同的初始化器有多个时
// 	class    *Struct   //变量的结构体
// 	function *Function //初始化方法；
// 	index    int       //初始化起可以排序；
// }

// func (initor *Initiator) GenerateCode(file *GenedFile) string {
// 	return "\n"
// }

type Initiators struct {
	list         map[string]*Variable //变量名,creator返回值中的名字。否则通过default命名；
	defaultValue *Variable            //默认变量的命名，以及默认变量的获取逻辑全部在
}

func createInitiators() *Initiators {
	return &Initiators{
		list:         make(map[string]*Variable),
		defaultValue: nil,
	}
}

func (inits *Initiators) addInitiator(initiator *Variable) {
	name := initiator.name
	if len(name) == 0 {
		name = "default_" + initiator.creator.pkg.Project.getRelativeModePath(initiator.creator.pkg.modPath) + "_" + initiator.class.Name
		initiator.name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
		if inits.defaultValue != nil && strings.HasPrefix(inits.defaultValue.name, "default_") {
			log.Fatalf("only one initiator can have empty name but %s in %s already decleaed when parse in %s",
				inits.defaultValue.name,
				inits.defaultValue.creator.goFile.path,
				initiator.creator.goFile.path,
			)
		}
		inits.defaultValue = initiator //没有名字的优先作为默认值
	}
	// 遇到的第一个初始化函数作为default值；后续如果有没有名字的，会替换；
	if inits.defaultValue == nil {
		inits.defaultValue = initiator
	}
	inits.list[name] = initiator
}
func (init *Initiators) getVariableName(name string) string {
	if variable, ok := init.list[name]; ok {
		return variable.name
	}
	//此处的defa
	return init.defaultValue.name
}
