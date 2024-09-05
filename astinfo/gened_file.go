package astinfo

import (
	"strconv"
	"strings"
)

// 每个有自动生成代码的package 会有一个GenedFile类；
type GenedFile struct {
	pkg *Package
	// for gen code
	genCodeImport        map[string]*Import //产生code时会引入其他模块的内容，此时每个模块需要一个名字；但是名字还不能重复
	genCodeImportNameMap map[string]int     //记录mode的个数；
}

func createGenedFile() GenedFile {
	return GenedFile{
		genCodeImport:        make(map[string]*Import),
		genCodeImportNameMap: make(map[string]int),
	}
}
func (file *GenedFile) getImport(modePath, modeName string) (result *Import) {
	if impt, ok := file.genCodeImport[modePath]; ok {
		return impt
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
		sb.WriteString(v.Name)
		sb.WriteString(" \"")
		sb.WriteString(v.Path)
		sb.WriteString("\"\n")
	}
	sb.WriteString(")\n")
	return sb.String()
}
