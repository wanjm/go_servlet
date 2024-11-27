package astinfo

import (
	"fmt"
	"strings"
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
	// initiatorMap map[*Struct]*Initiators //便于注入时根据类型存照
	project *Project
}

// 建立依赖关系树，并生成代码
func (manager *InitiatorManager) genInitiator() {
	// 获取所有的初始化函数
	for _, pkg := range manager.project.Package {
		pkg.genInitiator(manager)
	}
	// 生成所有的返回值变量
	leftLength := len(manager.dependNodes)
	lastlength := leftLength + 1
	level := 1
	for leftLength != lastlength {
		lastlength = leftLength
		manager.buildTree(level)
		level++
		leftLength = len(manager.dependNodes)
	}
	if leftLength > 0 {
		for _, node := range manager.dependNodes {
			fmt.Printf("can't find paramter for initiator %s\n", node.function.Name)
		}
		panic("can't find paramter for initiator")
	}
}
func (manager *InitiatorManager) checkReady(node *DependNode) bool {
	param := node.function.Params
	project := manager.project
	for _, p := range param {
		p.class = p.findStruct(true)
		if len(project.getVariable(p.class.(*Struct), p.name)) == 0 {
			return false
		}
	}
	manager.genVariable(node)
	return true
}

// 建立依赖关系树
// 目前采用简单for循环，找到依赖关系后再生成variable的方法，完成依赖关系的建立
func (manager *InitiatorManager) buildTree(level int) {
	// root := &manager.root
	c := 0
	for _, node := range manager.dependNodes {
		if manager.checkReady(node) {
			// root.children = append(root.children, node)
			node.level = level
		} else {
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
