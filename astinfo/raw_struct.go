package astinfo

type RawTypeInterface interface {
	getName() string
}
type RawType struct {
	Name string
}

func (r *RawType) getName() string {
	return r.Name
}

type ArrayType struct {
	OriginType interface{} //可以是结构体，interface，基本类型
}

func (a *ArrayType) getName() string {
	return "array"
}

type MapType struct {
}

func (m *MapType) getName() string {
	return "map"
}
