package astinfo

import "fmt"

type Variable struct {
	class        *Struct
	isPointer    bool
	name         string
	calledInFile *GenedFile //自己会被在哪个pkg中使用
}

func (variable *Variable) generateCode() string {
	impt := variable.calledInFile.getImport(variable.class.Package.modPath, variable.class.Package.modName)
	return fmt.Sprintf("%s.%s{}", impt.Name, variable.class.Name)
}
