package astinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 每个有自动生成代码的package 会有一个GenedFile类；
type GenedFile struct {
	// pkg *Package
	// for gen code
	name                 string             //文件名,without go;
	genCodeImport        map[string]*Import //产生code时会引入其他模块的内容，此时每个模块需要一个名字；但是名字还不能重复
	genCodeImportNameMap map[string]int     //记录mode的个数；
	contents             []*strings.Builder
}

func createGenedFile(fileName string) *GenedFile {
	return &GenedFile{
		name:                 fileName,
		genCodeImport:        make(map[string]*Import),
		genCodeImportNameMap: make(map[string]int),
	}
}

func (file *GenedFile) save() {
	if len(file.contents) == 0 {
		return
	}
	osfile, err := os.Create(file.name + ".go")
	if err != nil {

	}
	osfile.WriteString("package gen\n")
	osfile.WriteString(file.genImport())
	for _, content := range file.contents {
		osfile.WriteString(content.String())
	}
}
func (file *GenedFile) addBuilder(builder *strings.Builder) {
	file.contents = append(file.contents, builder)
}

// 根据modePath获取Import信息；理论上该函数不需要modeName，但是为了最大限度的代码可读性，还是带上了modeName；
func (file *GenedFile) getImport(modePath, modeName string) (result *Import) {
	if impt, ok := file.genCodeImport[modePath]; ok {
		return impt
	}
	// pkg的modName是在解析package代码时生成的。然后对于第三方的pkg，由于不会解析packge，所以其modeName为空，此时用modePath的baseName来代替，不会产生问题；
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
	for _, v := range file.genCodeImport {
		baseName := filepath.Base(v.Path)
		if baseName != v.Name {
			sb.WriteString(v.Name)
		}
		sb.WriteString(" \"")
		sb.WriteString(v.Path)
		sb.WriteString("\"\n")
	}
	sb.WriteString(")\n")
	return sb.String()
}
