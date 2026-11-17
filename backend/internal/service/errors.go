package service

import (
	"errors"
	"gorm.io/gorm"
	"time"
)

type Error struct {
	Status        int
	Code, Message string
	RetryAfter    time.Duration
}

func (e *Error) Error() string { return e.Message }
func Fail(status int, code, msg string) error {
	return &Error{Status: status, Code: code, Message: msg}
}
func Normalize(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Fail(404, "not_found", "记录不存在")
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return Fail(409, "duplicate", "名称已存在")
	}
	return err
}
