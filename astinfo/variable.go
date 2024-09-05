package astinfo

import "fmt"

type Variable struct {
	class        *Struct
	isPointer    bool
	name         string
	creator      *Function  //既可能是Function，也可能是Method
	calledInFile *GenedFile //自己会被在哪个pkg中使用
}

func (variable *Variable) generateCode(receiverPrefix string) string { //receiverPrefix是带.的
	if variable.creator != nil {
		return fmt.Sprintf(receiverPrefix + variable.creator.Name + "()")
	}
	impt := variable.calledInFile.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	return fmt.Sprintf("%s.%s{}", impt.Name, variable.class.Name)
}
