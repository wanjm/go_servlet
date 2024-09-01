package astinfo

import (
	"fmt"
	"strings"
)

type Struct struct {
	Name           string
	ImportUrl      string
	ServletMethods []*Method           //记录路由代码
	CreatorMethods map[*Struct]*Method //纪录构建默认参数的代码, key是构建的struct
	Import         *Import
	Package        *Package
	structFound    bool

	// 自动生成代码相关参数
	variableName string
}

type StructType struct {
	Struct    *Struct
	IsPointer bool
}

func CreateStruct(name string, pkg *Package) *Struct {
	return &Struct{
		Name:           name,
		Package:        pkg,
		CreatorMethods: make(map[*Struct]*Method),
	}
}

func (class *Struct) GenerateCode() string {
	class.variableName = class.Package.ModInfo.Name + class.Name
	var sb strings.Builder
	sb.WriteString(class.generateObject())
	for _, servlet := range class.ServletMethods {
		sb.WriteString(servlet.Receiver.GenerateCode())
	}
	return sb.String()
}

// 该方法会用于生成变量的代码；
// 1. 生成用于servet的类的对象；
// 2. 用于生成servlet参数的对象；
// 是否生成注入的代码，需要考虑 上述1，2的注入方法是否有区别
func (class *Struct) generateObject() string {
	// 变量名的规则是 ${modName}${struct.Name}
	codeFmt := "%s:= %s.%s{}\n"
	return fmt.Sprintf(codeFmt, class.variableName, class.Package.ModInfo.Name, class.Name)
}

func (class *Struct) addServlet(method *Method) {
	class.ServletMethods = append(class.ServletMethods, method)
}

func (class *Struct) addCreator(childClass *Struct, method *Method) {
	class.CreatorMethods[childClass] = method
}

func (class *Struct) GetCreatorCode4Struct(childClass *Struct) string {
	if method, ok := class.CreatorMethods[childClass]; ok {
		return method.Name + "()\n"
	} else {
		return childClass.generateObject()
	}
}
