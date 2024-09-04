package astinfo

import "strconv"

// 每个有自动生成代码的package 会有一个GenedFile类；
type GenedFile struct {
	pkg *Package
	// for gen code
	genCodeImport        map[string]*Import //产生code时会引入其他模块的内容，此时每个模块需要一个名字；但是名字还不能重复
	genCodeImportNameMap map[string]int     //记录mode的个数；
}

func (file *GenedFile) getImport(importPkg *Package) (result *Import) {
	modePath := importPkg.modPath
	if impt, ok := file.genCodeImport[modePath]; ok {
		return impt
	}
	if _, ok := file.genCodeImportNameMap[importPkg.modName]; ok {
		file.genCodeImportNameMap[importPkg.modName] = file.genCodeImportNameMap[importPkg.modName] + 1
		result = &Import{
			Name: importPkg.modName + strconv.Itoa(file.genCodeImportNameMap[importPkg.modName]),
			Path: modePath,
		}
	} else {
		file.genCodeImportNameMap[importPkg.modName] = 0
		result = &Import{
			Name: importPkg.modName,
			Path: modePath,
		}
	}
	file.genCodeImport[modePath] = result
	return
}
