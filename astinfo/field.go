package astinfo

import (
	"fmt"
	"go/ast"
	"strings"
)

type FieldComment struct {
	defaultValue string //记录该属性的默认值，在struct的field中有使用；
	// isRequired   bool   //记录该字段是否必须赋值，区别于gin的默认处理方法，必传表示在报文中必须存在
	validString string //校验变量是否符合要求的代码； $>10 && $<11
	comment     string
}

func (comment *FieldComment) dealValuePair(key, value string) {
	switch key {
	case "default":
		comment.defaultValue = value
	case "valid":
		comment.validString = value
	default:
		comment.comment = key
	}
}

// 变量名和变量类型的定义
// 用于函数的参数和返回值，struct的属性；

// FieldBasic 用于 FunctionField，VarField，Field；
// FunctionField用于Function和InterfaceField；
// InterfaceField用于Interface；
// VarField用于全局Var定义；
// Field用于Function的参数和返回值，以及Struct的属性；
type FieldBasic struct {
	Type    Typer // 实际可以为Struct，Interface， RawType
	Name    string
	Comment FieldComment
	// astRoot  *ast.Field
	GoSource *Gosourse //解析Filed时，其他type可能来源其他Package，此时需要Import内容来找到该包；
	parser   []Parser

	astDoc     *ast.CommentGroup // associated documentation; or nil
	astNames   []*ast.Ident      // value names (len(Names) > 0)
	astType    ast.Expr          // value type; or nil
	astComment *ast.CommentGroup // line comments; or nil
}

// genVariableCode
func (f *FieldBasic) GenVariableCode(goGenerated *GenedFile, wire bool) string {
	if f.Type == nil {
		fmt.Printf("skip gen variable for field %s as type is nil in %s\n", f.Name, f.GoSource.Path)
		return ""
	}
	variable := Variable{
		Type: f.Type,
		Name: f.Name,
		Wire: wire,
	}
	return variable.Generate(goGenerated)
}

func (field *Field) parseTag(fieldType *ast.BasicLit) {
	if fieldType != nil {
		tag := fieldType.Value
		tag = strings.Trim(tag, string(tag[0]))
		// if strings.Contains(tag, "wire") {
		// 	fmt.Print("hello")
		// }
		tagList := Fields(tag)
		for _, tag := range tagList {
			kv := strings.Split(tag, ":")
			if len(kv) == 2 {
				field.Tags[kv[0]] = strings.Trim(kv[1], "\"")
			}
		}
	}
}
func (field *FieldBasic) parseComment(fieldType *ast.CommentGroup) {
	if fieldType == nil || len(fieldType.List) <= 0 {
		return
	}
	content := strings.Trim(fieldType.List[0].Text, " /")
	parseValidComment(content, &field.Comment)
}

// Parse() error
// name type;
// name map
// name []arrays
// 此函数仅解析结构，然后在外面解析名字，拆分为多个Field
func (field *FieldBasic) Parse(typeMap map[string]*Field) error {
	fieldType := field.astType
	//field.Name="名字在调用本函数的外面解析，因为一个类型可能有多个名字，需要拆分为多个Field"
	field.ParseType(fieldType, typeMap)
	field.parseComment(field.astComment)

	return nil
}

// 在pkg内解析Type；
// 只有从结构体调入该函数时，typeMap才可能不为空；
// type TypeA[K,V] struct{****}
// type TypeB[K] K;
// type TypeC[K]=K;
// type TypeD[K] interface{***}
func parseType(fieldType ast.Expr, goSource *Gosourse, typeMap map[string]*Field) Typer {
	var resultType Typer
	switch fieldType := fieldType.(type) {
	case *ast.ArrayType:
		// 内置array类型
		// field的pkg指向原始类型；
		// field的class只想ArrayType;
		// ArrayType中的pkg，typeName，class指向具体的类型
		array := ArrayType{}
		resultType = &array
		array.Typer = parseType(fieldType.Elt, goSource, typeMap)
	case *ast.StarExpr:
		var pointer Typer
		pointer = parseType(fieldType.X, goSource, typeMap)
		resultType = NewPointerType(pointer)
	case *ast.Ident:
		resultType = goSource.getType(fieldType.Name, typeMap)
	case *ast.SelectorExpr:
		// 其他package的结构体，=》pkg1.Struct
		// field定义的selector，就只考虑pkg1
		pkgName := fieldType.X.(*ast.Ident).Name
		typeName := fieldType.Sel.Name
		// 解析import时，已经跳过了import C；
		// 但是解析Field时，还有可能是C，所以也要跳过；
		if pkgName != "C" {
			resultType = goSource.FindPackage(pkgName).GetTyper(typeName)
		}
	case *ast.MapType:
		mapType := MapType{}
		resultType = &mapType
	case *ast.InterfaceType:
		//匿名interface；
	case *ast.StructType:
		//匿名结构体
	case *ast.IndexListExpr:
		//var a className[k, v]
		// ast.IndexListExpr.X=>className;
		// ast.IndexListExpr.Indices=>[k, v];
		// className := fieldType.X.(*ast.Ident).Name
		// goSource.Pkg.FillType(className, &resultType)
	case *ast.FuncType:
	case *ast.ChanType:
	///...号参数在目前的解析情况下不会遇到；
	case *ast.IndexExpr:
		//atomic.Pointer[func()]
		//ast.IndexExpr.X=>atomic.Pointer;
		//ast.IndexExpr.Index=>func();
		//fmt.Printf("fieldType is nil in '%s' current not supported\n", goSource.Path)
	case *ast.ParenExpr:
		//onExit (func(interface{}))
		//fmt.Printf("fieldType is nil in '%s' current not supported\n", goSource.Path)
	case nil:
		fmt.Printf("fieldType is nil in '%s' current not supported\n", goSource.Path)
	default:
		// TODO: 需要添加日志级别，再打印日志
		fmt.Printf("unknown field type '%T' in '%s'\n", fieldType, goSource.Path)
		return nil
	}
	//如果将来Typer需要全局唯一，此处可以先找到唯一值，再赋值给typer；
	return resultType
}

func (field *FieldBasic) ParseType(fieldType ast.Expr, typeMap map[string]*Field) {
	field.Type = parseType(fieldType, field.GoSource, typeMap)
}

type Field struct {
	FieldBasic
	Tags   map[string]string
	astTag *ast.BasicLit
}

func (field *Field) Parse(typeMap map[string]*Field) error {
	field.parseTag(field.astTag)
	return field.FieldBasic.Parse(typeMap)
}
func (field *Field) GenNilCode(file *GenedFile) string {
	nt := field.Type
	if IsPointer(field.Type) {
		nt = GetBasicType(nt)
		res := "if a!=nil {\n" + field.genNilCode(nt, file) + "\n}\n"
		return res
	}
	return field.genNilCode(nt, file)
}

func (field *Field) genNilCode(nt Typer, file *GenedFile) string {
	switch nt := nt.(type) {
	case *Struct:
		return nt.GenNilCode(file)
	case *ArrayType:
		name := field.Type.RefName(file)
		return fmt.Sprintf("if *a == nil {\n*a = %s{}\n}", name)
	default:
		return ""
	}
}

func NewField(root *ast.Field, source *Gosourse) *Field {
	return &Field{
		FieldBasic: FieldBasic{
			GoSource:   source,
			astDoc:     root.Doc,
			astNames:   root.Names,
			astType:    root.Type,
			astComment: root.Comment,
		},
		astTag: root.Tag,
		Tags:   make(map[string]string),
	}
}
func NewSimpleField(typer Typer, name string) *Field {
	return &Field{
		FieldBasic: FieldBasic{
			Name: name,
			Type: typer,
		},
	}
}

type VarFieldHelper struct {
	varField FieldBasic
	astRoot  *ast.ValueSpec
	goSource *Gosourse
}

type VarField = FieldBasic

func NewVarFieldHelper(root *ast.ValueSpec, source *Gosourse) *VarFieldHelper {
	return &VarFieldHelper{
		astRoot:  root,
		goSource: source,
	}
}

func (v *VarFieldHelper) Parse() error {
	var root = v.astRoot
	var field = FieldBasic{
		GoSource:   v.goSource,
		astDoc:     root.Doc,
		astNames:   root.Names,
		astType:    root.Type,
		astComment: root.Comment,
	}
	// 目前仅支持解析普通var定于，用于完成rpc client的赋值；其他都忽略；
	if field.astType == nil {
		// TODO：添加级别日志；
		return nil
	}
	field.Parse(nil)
	if len(root.Names) != 0 {
		for _, name := range root.Names {
			field1 := field
			field1.Name = name.Name
			v.goSource.Pkg.GlobalVar[name.Name] = &field1
		}
	}
	return nil
}

func FieldListToMap(fields []*Field) map[string]*Field {
	res := make(map[string]*Field)
	for _, field := range fields {
		res[field.Name] = field
	}
	return res
}
