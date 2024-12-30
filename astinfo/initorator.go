package astinfo

import (
	"fmt"
	"strings"
)

const (
	globalPrefix = "__global_"
)

// 初始化函数依赖关系节点
type DependNode struct {
	level          int
	children       []*DependNode //依赖于自己的节点
	parent         []*DependNode //自己依赖的节点
	function       *Function
	returnVariable *Variable
}
type InitiatorManager struct {
	root        DependNode
	dependNodes []*DependNode
	sortedNodes []*DependNode
	// initiatorMap map[*Struct]*Initiators //便于注入时根据类型存照
	project *Project
}

// 1. 收集所有的initiator函数到一个数组中；
// 2. 根据依赖关系生成变量名，只有一个函数的所有依赖值都存在时，才生成变量名；
// 3. 生成变量名时，同时记录依赖层级；
// 4. 当有变量无法程程时，则表示依赖条件不满足，a: 依赖的变量不存在， b：存在相互依赖，无法生成变量名；
func (manager *InitiatorManager) genInitiator() {
	// 获取所有的初始化函数
	for _, pkg := range manager.project.Package {
		pkg.genInitiator(manager)
	}
	// 生成所有的返回值变量
	leftLength := len(manager.dependNodes)
	lastlength := leftLength + 1
	for leftLength != lastlength {
		lastlength = leftLength
		manager.buildTree()
		leftLength = len(manager.dependNodes)
	}
	if leftLength > 0 {
		for _, node := range manager.dependNodes {
			fmt.Printf("can't find paramter for initiator %s\n", node.function.Name)
		}
		panic("can't find paramter for initiator")
	}
}

func (manager *InitiatorManager) genInitiatorCode() {
	var file *GenedFile = createGenedFile("goservlet_initiator")
	define := strings.Builder{}
	assign := strings.Builder{}
	file.addBuilder(&define)
	file.addBuilder(&assign)
	assign.WriteString("func initVariable() {\n")
	for _, node := range manager.sortedNodes {
		initor := node.function
		variable := node.returnVariable
		if variable != nil {
			define.WriteString(variable.genDefinition(file))
			define.WriteString("\n")
			assign.WriteString(variable.name)
			assign.WriteString("=")
		}
		assign.WriteString(initor.genCallCode("", file))
		assign.WriteString("\n")
	}
	assign.WriteString("}\n")
	file.save()
	manager.project.addInitFuncs("initVariable()")
}
func (manager *InitiatorManager) checkReady(node *DependNode) bool {
	param := node.function.Params
	project := manager.project
	level := 0
	for _, p := range param {
		p.class = p.findStruct(true)
		dep := project.getDependNode(p.class.(*Struct), p.name)
		if dep == nil {
			return false
		}
		if level < dep.level {
			level = dep.level
		}
	}
	node.level = level + 1
	manager.sortedNodes = append(manager.sortedNodes, node)
	manager.genVariable(node)
	return true
}

// 建立依赖关系树
// 目前采用简单for循环，找到依赖关系后再生成variable的方法，完成依赖关系的建立
func (manager *InitiatorManager) buildTree() {
	// root := &manager.root
	c := 0
	for _, node := range manager.dependNodes {
		if !manager.checkReady(node) {
			manager.dependNodes[c] = node
			c++
		}
	}
	manager.dependNodes = manager.dependNodes[:c]
}

func (manager *Project) addInitiatorVaiable(node *DependNode) {
	initiator := node.returnVariable
	// 后续添加排序功能
	// funcManager.initiator = append(funcManager.initiator, initiator)
	var inits *Initiators
	var ok bool
	if inits, ok = manager.initiatorMap[initiator.class]; !ok {
		inits = createInitiators()
		manager.initiatorMap[initiator.class] = inits
	}
	inits.addInitiator(node)
}

func (manager *InitiatorManager) genVariable(dependNode *DependNode) {
	if len(dependNode.function.Results) == 0 {
		return
	}
	result := dependNode.function.Results[0]
	//  := initor.Results[0]
	name := result.name
	if len(name) == 0 {
		name = strings.ReplaceAll(result.pkg.modPath, ".", "_")
		name = strings.ReplaceAll(name, "/", "_")
	}
	name = globalPrefix + name
	variable := Variable{
		// creator:   initor,
		class:     result.findStruct(true),
		name:      name,
		isPointer: result.isPointer,
	}
	dependNode.returnVariable = &variable
	manager.project.addInitiatorVaiable(dependNode)
}

func (pkg *Package) genInitiator(manager *InitiatorManager) {
	manager.dependNodes = append(manager.dependNodes, pkg.initiators...)
	for _, class := range pkg.StructMap {
		class.genInitiator(manager)
	}
}
func (class *Struct) genInitiator(manager *InitiatorManager) {
	manager.dependNodes = append(manager.dependNodes, class.initiators...)
}
