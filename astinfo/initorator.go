package astinfo

import "strings"

// 初始化函数依赖关系节点
type DependNode struct {
	level          int
	children       []*DependNode //依赖于自己的节点
	parent         []*DependNode //自己依赖的节点
	function       *Function
	returnVariable *Variable
}
type InitiatorManager struct {
	root         DependNode
	dependNodes  []*DependNode
	initiatorMap map[*Struct]*Initiators //便于注入时根据类型存照
	project      *Project
}

// 建立依赖关系树，并生成代码
func (manager *InitiatorManager) genInitiator() {
	// 获取所有的初始化函数
	for _, pkg := range manager.project.Package {
		pkg.genInitiator(manager)
	}
	// 生成所有的返回值变量
	for _, dependNode := range manager.dependNodes {
		manager.genVariable(dependNode)
	}
}

// 建立依赖关系树
func (manager *InitiatorManager) buildTree() {
	if len(manager.dependNodes) == 0 {
		return
	}
	root := &manager.root
	c := 0
	for i, l := 0, len(manager.dependNodes); i < l; i++ {
		node := manager.dependNodes[i]
		if len(node.function.Params) == 0 {
			root.children = append(root.children, node)
			node.level = 1
		}
		if i != c {
			manager.dependNodes[c] = node
			c++
		}
	}
	manager.dependNodes = manager.dependNodes[:c]
}

func (manager *InitiatorManager) addInitiatorVaiable(node *DependNode) {
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
	manager.addInitiatorVaiable(dependNode)
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
