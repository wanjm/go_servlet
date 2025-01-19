package astinfo

import (
	"fmt"
)

// 定义了一个creator的返回值，用于创建变量；
// 将来Field的注入等Variable来完成
type Variable struct {
	class     *Struct
	isPointer bool
	name      string
	creator   *Function //既可能是Function，也可能是Method
}

// 生成代码的三种场景
// 从全局变量获取
// reciver.function creator!=nil, receiverPrex!=""
// schema.struct
// schema.function  creator!=nil, receiverPrefix==""
// 返回值无\n
func (variable *Variable) generateCode(receiverPrefix string, file *GenedFile) string { //receiverPrefix是带.的
	creator := variable.creator
	if creator == nil {
		//如果没有自带构造器，则先从全局变量中寻找, 全部变量目前支持指针和interface，但是此处没有做检查
		// if variable.isPointer {
		// 生成注入代码时，应该走这里；
		name := variable.class.Package.Project.getVariableName(variable.class, variable.name)
		if len(name) > 0 {
			return name
		}
		// }
		creator = variable.class.getCreator(variable.class)
		if creator != nil {
			variable.creator = creator
			variable.isPointer = creator.Results[0].isPointer
		}
	}
	if creator != nil {
		return creator.genCallCode(receiverPrefix, file)
	}
	objPrefix := ""
	if variable.isPointer {
		objPrefix = "&"
	}
	return objPrefix + variable.class.generateConstructCode(file)

}

// 返回值无\n
func (variable *Variable) genDefinition(file *GenedFile) string {
	impt := file.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	pointerMark := ""
	if variable.isPointer {
		pointerMark = "*"
	}
	return fmt.Sprintf("var %s %s%s.%s", variable.name, pointerMark, impt.Name, variable.class.Name)
}
