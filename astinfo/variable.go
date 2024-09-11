package astinfo

import (
	"fmt"
	"strings"
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
		//如果没有自带构造器，则先从全局变量寻找, 全局变量仅支持指针
		if variable.isPointer {
			name := variable.class.Package.Project.getVariable(variable.class, variable.name)
			if len(name) > 0 {
				return name
			}
		}
		creator := variable.class.getCreator(variable.class)
		if creator != nil {
			variable.creator = creator
			variable.isPointer = creator.Results[0].isPointer
		}
	}
	if creator != nil {
		var prefix string
		if len(receiverPrefix) > 0 {
			prefix = receiverPrefix
		} else {
			pkg := creator.pkg
			impt := file.getImport(pkg.modPath, pkg.modName)
			prefix = impt.Name + "."
		}
		return fmt.Sprintf(prefix + creator.Name + "()")
	}

	impt := file.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	fieldsValue := make([]string, len(variable.class.fields))
	for index, field := range variable.class.fields {
		childVar := Variable{
			class:     field.class,
			isPointer: field.isPointer,
			name:      field.name,
		}
		fieldsValue[index] = field.name + ":" + childVar.generateCode("", file)
	}
	body := strings.Join(fieldsValue, ",\n")
	objPrefix := ""
	if variable.isPointer {
		objPrefix = "&"
	}
	return fmt.Sprintf("%s%s.%s{%s}", objPrefix, impt.Name, variable.class.Name, body)
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