package service

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)

func validateUsername(s string) error {
	if !usernamePattern.MatchString(s) {
		return Fail(400, "validation", "账号须为 3–32 位字母、数字、点、下划线或短横线")
	}
	return nil
}
func validatePassword(s string) error {
	if utf8.RuneCountInString(s) < 8 || len(s) > 72 {
		return Fail(400, "validation", "密码至少 8 个字符，最多 72 字节")
	}
	return nil
}
func validateDisplay(s string) error {
	if utf8.RuneCountInString(s) > 64 {
		return Fail(400, "validation", "姓名最多 64 个字符")
	}
	return nil
}
func validateRole(in RoleInput) error {
	if strings.TrimSpace(in.Name) == "" || utf8.RuneCountInString(in.Name) > 64 || utf8.RuneCountInString(in.Description) > 500 {
		return Fail(400, "validation", "角色名称为 1–64 个字符，说明最多 500 个字符")
	}
	return nil
}
