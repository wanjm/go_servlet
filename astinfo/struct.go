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

	// 自动生成代码相关参数，此处可能需要更改为StructObject对象
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

// 注意跟变量注入区分开来
func (class *Struct) GenerateCode() string {
	class.variableName = class.Package.ModInfo.Name + class.Name
	var sb strings.Builder
	sb.WriteString(class.variableName + ":=" + class.generateObject())
	for _, servlet := range class.ServletMethods {
		sb.WriteString(servlet.GenerateCode())
	}
	return sb.String()
}

// 该方法会用于生成变量的代码；
// 1. 生成用于servet的类的对象；
// 2. 用于生成servlet参数的对象；
// 是否生成注入的代码，需要考虑 上述1，2的注入方法是否有区别
func (class *Struct) generateObject() string {
	// 变量名的规则是 ${modName}${struct.Name}
	codeFmt := "%s.%s{}\n"
	return fmt.Sprintf(codeFmt, class.Package.ModInfo.Name, class.Name)
}

func (class *Struct) addServlet(method *Method) {
	class.ServletMethods = append(class.ServletMethods, method)
}

// creator是用户提供的创建某个对象的方法，主要是用于设置请求的默认值；主要用于构建servlet的request参数
func (class *Struct) addCreator(childClass *Struct, method *Method) {
	class.CreatorMethods[childClass] = method
}

// 一个servlet的request对象，可以直接构造空方法，也可以调用该类型提供的creator方法；
func (class *Struct) GetCreatorCode4Struct(childClass *Struct) string {
	if method, ok := class.CreatorMethods[childClass]; ok {
		// 调用cratetor方法，则为该对象的变量+creator方法
		return class.variableName + "." + method.Name + "()\n"
	} else {
		// 直接构建空对象
		return childClass.generateObject()
	}
}
