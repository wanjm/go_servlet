package astinfo

type RawType struct {
	Name string
}

type ArrayType struct {
	// 为了先遇到结构体， 后定义的问题，这里先定义了typeName和pkg；
	// 后续跟Field结构体同步处理
	// 如果能给OriginType赋值，就不需要这两个字段了
	typeName   string     //类型名
	pkg        *Package   //class所在的包
	OriginType SchemaType //可以是结构体，interface，基本类型
}

type MapType struct {
}
