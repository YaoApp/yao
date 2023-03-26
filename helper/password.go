package helper

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"golang.org/x/crypto/bcrypt"
)

// PasswordValidate 校验密码
func PasswordValidate(password string, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		exception.New("密码不正确", 400).Throw()
		return false
	}
	return true
}

// ProcessPasswordValidate xiang.helper.PasswordValidate 校验密码
func ProcessPasswordValidate(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	return PasswordValidate(process.ArgsString(0), process.ArgsString(1))
}
