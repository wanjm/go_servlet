package astinfo

import (
	"sort"
	"unicode"
)

func firstLower(word string) string {
	return string(unicode.ToLower([]rune(word)[0])) + word[1:]
}
func getSortedKey[Value any](m map[string]Value) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
