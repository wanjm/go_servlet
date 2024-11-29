package astinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go/format"
)

// 每个有自动生成代码的package 会有一个GenedFile类；
type GenedFile struct {
	// pkg *Package
	// for gen code
	name                 string             //文件名,没有go后缀
	genCodeImport        map[string]*Import //产生code时会引入其他模块的内容，此时每个模块需要一个名字；但是名字还不能重复
	genCodeImportNameMap map[string]int     //记录mode的个数；
	contents             []*strings.Builder //本文件内容的多个片段，参见save函数
}

func createGenedFile(fileName string) *GenedFile {
	return &GenedFile{
		name:                 fileName,
		genCodeImport:        make(map[string]*Import),
		genCodeImportNameMap: make(map[string]int),
	}
}

// 保存文件
// 生成package语句
// 生成import语句
// 按照file.contents的顺序，生成文件内容
func (file *GenedFile) save() {
	if len(file.contents) == 0 {
		return
	}
	content := strings.Builder{}
	content.WriteString("package gen\n")
	content.WriteString(file.genImport())
	for _, content1 := range file.contents {
		content.WriteString(content1.String())
	}
	src, err := format.Source([]byte(content.String()))
	if err != nil {
		panic(err)
	}
	osfile, err := os.Create(file.name + ".go")
	if err != nil {
		panic(err)
	}
	osfile.Write(src)
}

func (file *GenedFile) addBuilder(builder *strings.Builder) {
	file.contents = append(file.contents, builder)
}

// 根据modePath获取Import信息；理论上该函数不需要modeName，但是为了最大限度的代码可读性，还是带上了modeName；
func (file *GenedFile) getImport(modePath, modeName string) (result *Import) {
	if impt, ok := file.genCodeImport[modePath]; ok {
		return impt
	}
	// pkg的modName是在解析package代码时生成的。然后对于第三方的pkg，由于不会解析packge，所以其modeName为空，
	// 此时用modePath的baseName来代替，会产生问题，并不是每个package的modeName都是baseName的。如"github.com/redis/go-redis/v9"的modeName是redis
	// 同时在生成代码时，会将baseModePath和modeName相同的，省掉modeName不写；但是go默认modeName是定义package时的packge字段描述的。
	// 此处使用等价规则，会产生问题；而由于我们不扫描第三方package，所以不知道其正确的modeName
	// 临时解决方案是，产生代码时，import全部写modeName，不省略；
	if len(modeName) == 0 {
		modeName = filepath.Base(modePath)
	}
	if _, ok := file.genCodeImportNameMap[modeName]; ok {
		file.genCodeImportNameMap[modeName] = file.genCodeImportNameMap[modeName] + 1
		result = &Import{
			Name: modeName + strconv.Itoa(file.genCodeImportNameMap[modeName]),
			Path: modePath,
		}
	} else {
		file.genCodeImportNameMap[modeName] = 0
		result = &Import{
			Name: modeName,
			Path: modePath,
		}
	}
	file.genCodeImport[modePath] = result
	return
}
func (file *GenedFile) genImport() string {
	if len(file.genCodeImport) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("import (\n")
	imports := make([]string, len(file.genCodeImport))
	var i = 0
	for _, v := range file.genCodeImport {
		// baseName := filepath.Base(v.Path)
		imports[i] = v.Name + " \"" + strings.ReplaceAll(v.Path, "\\", "/") + "\""
		/*
			// if baseName != v.Name {
			sb.WriteString(v.Name)
			// }
			_ = baseName
			sb.WriteString(" \"")
			sb.WriteString(strings.ReplaceAll(v.Path, "\\", "/"))
			sb.WriteString("\"\n")
		*/
		i++
	}
	sb.WriteString(strings.Join(imports, "\n"))
	sb.WriteString("\n)\n")
	return sb.String()
}
