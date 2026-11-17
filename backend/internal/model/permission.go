package model

type Permission struct {
	Code        string `json:"code" gorm:"primaryKey"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var Permissions = []Permission{
	{"users:read", "查看用户", "查看用户列表和用户资料"},
	{"users:manage", "管理用户", "创建、编辑、停用用户及重设密码，包含查看权限"},
	{"roles:read", "查看角色", "查看角色与权限清单"},
	{"roles:manage", "管理角色", "创建、编辑和删除角色，包含查看权限"},
	{"ai:use", "使用 AI", "自行生成密钥并调用 AI 服务"},
}
