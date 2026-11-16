package service

import (
	"errors"
	"gorm.io/gorm"
)

type Error struct {
	Status        int
	Code, Message string
}

func (e *Error) Error() string                { return e.Message }
func Fail(status int, code, msg string) error { return &Error{status, code, msg} }
func Normalize(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Fail(404, "not_found", "记录不存在")
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return Fail(409, "duplicate", "名称已存在")
	}
	return err
}
