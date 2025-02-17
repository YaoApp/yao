package helper

// 检查字符串数组中是否包含某个字符
func ContainsString(arr []string, char string) bool {
	for _, str := range arr {
		if str == char {
			return true
		}
	}
	return false
}
