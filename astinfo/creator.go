package astinfo

// type Initiator struct {
// 	name     string    //用于被初始化变量的名字；当相同的初始化器有多个时
// 	class    *Struct   //变量的结构体
// 	function *Function //初始化方法；
// 	index    int       //初始化起可以排序；
// }

// func (initor *Initiator) GenerateCode(file *GenedFile) string {
// 	return "\n"
// }

type Initiators struct {
	list         map[string]*Variable //变量名,creator返回值中的名字。否则通过default命名；
	defaultValue *Variable
}

func createInitiators() *Initiators {
	return &Initiators{
		list:         make(map[string]*Variable),
		defaultValue: nil,
	}
}
