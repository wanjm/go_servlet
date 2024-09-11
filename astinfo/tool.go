package astinfo

import "unicode"

func firstLower(word string) string {
	return string(unicode.ToLower([]rune(word)[0])) + word[1:]
}
