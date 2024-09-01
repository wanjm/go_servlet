package astinfo

type Projector interface {
	MyProject() *Project
}

type Namer interface {
	Name() string
}

type ImportPathProvider interface {
	ImportPath() string
}

type CodeGenerator interface {
	GenerateCode() string
}
