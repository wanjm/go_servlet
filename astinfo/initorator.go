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

func (manager *InitiatorManager) addInitiatorVaiable(initiator *Variable) {
	// 后续添加排序功能
	// funcManager.initiator = append(funcManager.initiator, initiator)
	var inits *Initiators
	var ok bool
	if inits, ok = manager.initiatorMap[initiator.class]; !ok {
		inits = createInitiators()
		manager.initiatorMap[initiator.class] = inits
	}
	inits.addInitiator(initiator)
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
	manager.addInitiatorVaiable(&variable)
}

func (pkg *Package) genInitiator(manager *InitiatorManager) {
	for _, function := range pkg.initiators {
		dependNode := &DependNode{
			function: function,
		}
		manager.dependNodes = append(manager.dependNodes, dependNode)
	}
	for _, class := range pkg.StructMap {
		class.genInitiator(manager)
	}
}
func (class *Struct) genInitiator(manager *InitiatorManager) {
	for _, function := range class.initiators {
		dependNode := &DependNode{
			function: function,
		}
		manager.dependNodes = append(manager.dependNodes, dependNode)
	}
}
