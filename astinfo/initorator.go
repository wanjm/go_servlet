package astinfo

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

const (
	globalPrefix = "__global_"
)

type VariableGenerator interface {
	RequiredFields() []*Field
	GeneredFields() []*Field
	GenerateDependcyCode(goGenerated *GenedFile) string
}

// 初始化函数依赖关系节点
type DependNode struct {
	Generator          VariableGenerator
	Parent             []*DependNode
	returnVariableName string
}

// getReturnName 获取返回值名称
func (d *DependNode) getReturnName() string {
	//此函数没有判空，是要求其他逻辑保证调用不到该函数。所以暂时不判空。如果有一天觉得无法保证。还是要改的；
	return d.getReturnField().Name
}

// getReturnField 获取返回值字段
func (d *DependNode) getReturnField() *Field {
	fields := d.Generator.GeneredFields()
	if len(fields) == 0 {
		return nil
	}
	return fields[0]
}

type InitGroup struct {
	Initorators []*DependNode
	Default     *DependNode
}

// addNode 添加节点
func (g *InitGroup) addNode(node *DependNode) {
	g.Initorators = append(g.Initorators, node)
	if g.Default == nil {
		g.Default = node
	} else if node.getReturnName() == "" {
		if g.Default.getReturnName() == "" {
			// 这里无法获取函数名，暂时注释掉
			fmt.Printf("more than one function return the same type %s,but without name\n", g.Default.getReturnField().Type.IDName())
		} else {
			g.Default = node
		}
	}
}

type InitManager struct {
	variableMap VariableMap //存放已经准备好了变量对象；
	readyNode   []*DependNode
	project     *MainProject
	nameValue   map[string]string // 用于生成nameValue map[string]any代码的map
}

// Generate(goGenerated *GenedFile) error
func (im *InitManager) Generate(goGenerated *GenedFile) error {
	if len(im.readyNode) == 0 {
		return nil
	}
	var definition strings.Builder
	var call strings.Builder
	definition.WriteString("var (\n")
	call.WriteString("func initVariable() {\n")
	for _, node := range im.readyNode {
		if node.returnVariableName != "" {
			definition.WriteString(fmt.Sprintf("%s %s\n", node.returnVariableName, node.getReturnField().Type.RefName(goGenerated)))
			call.WriteString(fmt.Sprintf("%s = ", node.returnVariableName))
		}
		call.WriteString(node.Generator.GenerateDependcyCode(goGenerated))
		call.WriteString("\n")
	}
	definition.WriteString(")\n")
	call.WriteString("}\n")
	goGenerated.AddBuilder(&definition)
	goGenerated.AddBuilder(&call)
	im.project.InitFuncs4All = append(im.project.InitFuncs4All, "initVariable")
	return nil
}

// GenterateTestCode 生成测试代码
func (im *InitManager) GenterateTestCode(goGenerated *GenedFile) {
	goGenerated.GetImport(SimplePackage("reflect", "reflect"))
	var testCode strings.Builder
	textTemplate := `
var nameValue map[string]interface{}
var typeValue map[reflect.Type]interface{}

func GetValue(value any) {
	// 检查是否为指针类型（否则无法设置值）
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return // 不是指针或指针为nil，无法设置值
	}
	// 获取指针指向的元素类型
	t := val.Elem().Type()
	// 查找对应的值
	if v, ok := typeValue[t]; ok {
		// 设置值
		val.Elem().Set(reflect.ValueOf(v))
	}
}
// GetValueByName
func GetValueByName(name string) any {
	return nameValue[name]
}
func PrepareTest() {
	Prepare()
	typeValue = make(map[reflect.Type]interface{})
	nameValue = make(map[string]interface{})
	{{range $username, $value := .NameValue}}	
    	nameValue["{{$username}}"] = {{$value}}
	{{end}}
	{{range $value := .TypeValue}}	
    	typeValue[reflect.TypeOf({{$value}})] = {{$value}}
	{{end}}
}`

	tmpl, err := template.New("test").Parse(textTemplate)
	if err != nil {
		panic(err)
	}
	var typeValue []string
	for _, v := range im.variableMap {
		typeValue = append(typeValue, v.Default.returnVariableName)
	}
	err = tmpl.Execute(&testCode, struct {
		NameValue map[string]string
		TypeValue []string
	}{
		NameValue: im.nameValue,
		TypeValue: typeValue,
	})
	if err != nil {
		panic(err)
	}
	goGenerated.AddBuilder(&testCode)
}

func (mp *MainProject) InitInitorator() {
	mp.InitManager = &InitManager{
		variableMap: make(map[string]*InitGroup),
		project:     mp,
		nameValue:   make(map[string]string),
	}
	mp.InitManager.initInitorator()
}

// 返回初始化函数和map，key为Typer，value为相同返回值的数组
func (im *InitManager) collect() ([]*DependNode, VariableMap) {
	p := im.project
	dependNode := []*DependNode{}
	var waittingVariableMap VariableMap = make(map[string]*InitGroup)
	// 收集initiator到functions中；
	// 建立候选变量map
	for _, pkg := range p.Packages {
		for _, function := range pkg.Initiator {
			node := waittingVariableMap.addVGenerator(function)
			dependNode = append(dependNode, node)
		}
		for _, class := range pkg.Structs {
			if class.Comment.AutoGen {
				exist := waittingVariableMap.getVariable(class, "")
				// 自动生成代码如果遇到其他有名字的，则放弃；
				if exist != nil && exist.getReturnName() == "" {
					continue
				}
				node := waittingVariableMap.addVGenerator(class)
				dependNode = append(dependNode, node)
			}
		}
	}
	return dependNode, waittingVariableMap
}

// initInitorator 初始化初始化函数
func (im *InitManager) initInitorator() {
	// 创建variableMap
	var globalIndex int = 0
	functions, waittingVariableMap := im.collect()
	//将所有节点连接到父节点
	for _, node := range functions {
		im.initParent(node, waittingVariableMap)
	}
	// 每轮从functions中取出已经准备好了的function，放到ready的function中；
	var found bool = true
	for found {
		found = false
		var index int = 0
		for _, node := range functions {
			if im.variableMap.checkReady(node) {
				if node.getReturnField() != nil {
					var realName = node.getReturnName()
					node.returnVariableName = globalPrefix + realName + "_" + strconv.Itoa(globalIndex)
					// 如果realName为空，则覆盖，因为原本就没有计划要；
					im.nameValue[realName] = node.returnVariableName
					globalIndex++
				}
				found = true
				im.variableMap.addNode(node)
				im.readyNode = append(im.readyNode, node)
			} else {
				functions[index] = node
				index++
			}
		}
		functions = functions[:index]
	}
	if len(functions) > 0 {
		// for _, node := range functions {
		// 	fmt.Printf("can't init function\n")
		// }
	}
}

// initParent 初始化父节点
func (im *InitManager) initParent(node *DependNode, waittingVariableMap VariableMap) {
	params := node.Generator.RequiredFields()
	for _, param := range params {
		parent := waittingVariableMap.getVariable(param.Type, param.Name)
		if parent != nil {
			node.Parent = append(node.Parent, parent)
		} else {
			fmt.Printf("can't init field: %s not found for type %s\n", param.Name, param.Type.IDName())
		}
	}
}

func (mp *MainProject) GetVariableName(typer Typer, name string) string {
	return mp.InitManager.variableMap.getVariable(typer, name).returnVariableName
}

func (mp *MainProject) GetVariableNode(typer Typer, name string) *DependNode {
	name = FirstLower(name)
	return mp.InitManager.variableMap.getVariable(typer, name)
}

type VariableMap map[string]*InitGroup //key是原始类型的名字"int"，"pkg.Struct"

func (vm VariableMap) getVariable(typer Typer, name string) *DependNode {
	group := vm[typer.IDName()]
	if group == nil {
		return nil
	}
	for _, initorator := range group.Initorators {
		if initorator.getReturnName() == name {
			return initorator
		}
	}
	return group.Default
}

// addVGenerator 添加初始化函数
func (im VariableMap) addVGenerator(function VariableGenerator) *DependNode {
	var node = &DependNode{
		Generator: function,
	}
	im.addNode(node)
	return node
}

// addNode 添加节点
func (im VariableMap) addNode(node *DependNode) {
	returnField := node.getReturnField()
	if returnField == nil {
		return
	}
	typer := returnField.Type
	group := im[typer.IDName()]
	if group == nil {
		group = &InitGroup{
			Initorators: []*DependNode{},
		}
		im[typer.IDName()] = group
	}
	group.addNode(node)
}

// checkReady 检查是否所有父节点都已初始化
func (im VariableMap) checkReady(node *DependNode) bool {
	for _, parent := range node.Parent {
		if parent.returnVariableName == "" {
			return false
		}
	}
	return true
}
