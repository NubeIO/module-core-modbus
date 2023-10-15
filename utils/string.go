package utils

import "strings"

func IsEqualIgnoreCase(str string, args ...string) bool {
	for _, s := range args {
		if strings.ToUpper(str) == strings.ToUpper(s) {
			return true
		}
	}
	return false
}
