package astinfo

import "fmt"

type Variable struct {
	class     *Struct
	isPointer bool
	name      string
	creator   *Function //既可能是Function，也可能是Method
}

// 生成代码的三种场景
// reciver.function creator!=nil, receiverPrex!=""
// schema.struct
// schema.function  creator!=nil, receiverPrefix==""
// 返回值无\n
func (variable *Variable) generateCode(receiverPrefix string, file *GenedFile) string { //receiverPrefix是带.的
	if variable.creator != nil {
		var prefix string
		if len(receiverPrefix) > 0 {
			prefix = receiverPrefix
		} else {
			pkg := variable.creator.pkg
			impt := file.getImport(pkg.modPath, pkg.modName)
			prefix = impt.Name + "."
		}
		return fmt.Sprintf(prefix + variable.creator.Name + "()")
	}
	impt := file.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	return fmt.Sprintf("%s.%s{}", impt.Name, variable.class.Name)
}

// 返回值无\n
func (variable *Variable) genDefinition(file *GenedFile) string {
	impt := file.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	return fmt.Sprintf("var %s %s.%s", variable.name, impt.Name, variable.class.Name)
}
