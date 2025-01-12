package astinfo

type RawType struct {
	Name string
}

type ArrayType struct {
	// 为了先遇到结构体， 后定义的问题，这里先定义了typeName和pkg；
	// 后续跟Field结构体同步处理
	// 如果能给OriginType赋值，就不需要这两个字段了
	typeName  string     //类型名
	pkg       *Package   //class所在的包
	class     SchemaType //可以是结构体，interface，基本类型
	isPointer bool       //array中的子类型是否是指针 []*string
}

func (r *ArrayType) GetTypename() string {
	return "array"
}
func (field *ArrayType) findStruct(force bool) *Struct {
	// 此处如果代码错误，会出现class为Interface，但是强转为Struct的情况，让程序报错
	class := field.pkg.getStruct(field.typeName, force)
	return class
}

func (r *RawType) GetTypename() string {
	return r.Name
}

type MapType struct {
}

func (r *MapType) GetTypename() string {
	return "map"
}

type EmptyType struct {
}
