package astinfo

type Variable struct {
	class     *Struct
	isPointer bool
	name      string
}
