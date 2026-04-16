package models

import "encoding/json"

// Permission 权限结构定义
// 用于定义系统中所有的权限项
type Permission struct {
	Key      string `json:"key"`      // 权限标识，使用点号分隔
	Name     string `json:"name"`     // 权限中文名称
	Category string `json:"category"` // 权限所属分组
}

// AllPermissions 系统所有权限清单
// 定义了系统中可用的所有权限项，权限标识使用点号分隔
var AllPermissions = []Permission{
	// 系统权限
	{Key: "dashboard", Name: "仪表盘", Category: "系统"},
	{Key: "user.manage", Name: "用户管理", Category: "系统"},
	{Key: "audit.view", Name: "查看审计", Category: "系统"},

	// 合同权限
	{Key: "contract.read", Name: "查看合同", Category: "合同"},
	{Key: "contract.create", Name: "创建合同", Category: "合同"},
	{Key: "contract.edit", Name: "编辑合同", Category: "合同"},
	{Key: "contract.delete", Name: "删除合同", Category: "合同"},

	// 客户权限
	{Key: "customer.read", Name: "查看客户", Category: "客户"},
	{Key: "customer.create", Name: "创建客户", Category: "客户"},
	{Key: "customer.edit", Name: "编辑客户", Category: "客户"},
	{Key: "customer.delete", Name: "删除客户", Category: "客户"},

	// 合同类型权限
	{Key: "contract_type.manage", Name: "管理合同类型", Category: "合同"},

	// 审批权限
	{Key: "approval.process", Name: "审批处理", Category: "审批"},
	{Key: "approval.view", Name: "查看审批", Category: "审批"},
}

// RolePermissionMap 角色默认权限映射
// 定义每个角色默认拥有的权限列表
var RolePermissionMap = map[string][]string{
	"admin":            {"all"},                                                                                                                                                                              // 超级管理员拥有所有权限
	"manager":          {"dashboard", "contract.read", "contract.create", "contract.edit", "contract_type.manage", "customer.read", "customer.create", "customer.edit", "approval.process", "approval.view"}, // 经理
	"user":             {"dashboard", "contract.read", "contract.create", "contract.edit", "contract.delete", "contract_type.manage", "customer.read", "customer.create", "approval.view"},                   // 销售
	"sales_director":   {"dashboard", "contract.read", "contract.create", "contract.edit", "contract_type.manage", "customer.read", "customer.create", "approval.process", "approval.view"},                  // 销售总监
	"tech_director":    {"dashboard", "contract.read", "contract.create", "contract.edit", "contract_type.manage", "customer.read", "customer.create", "approval.process", "approval.view"},                  // 技术总监
	"finance_director": {"dashboard", "contract.read", "contract.create", "contract.edit", "contract_type.manage", "customer.read", "customer.create", "approval.process", "approval.view"},                  // 财务总监
	"contract_admin":   {"dashboard", "contract.read", "contract.create", "contract.edit", "contract_type.manage", "customer.read", "customer.create", "customer.edit", "approval.process", "approval.view"}, // 合同管理员
	"sales":            {"dashboard", "contract.read", "contract.create", "contract.edit", "contract.delete", "contract_type.manage", "customer.read", "customer.create", "approval.view"},                   // 销售人员
	"audit_admin":      {"dashboard", "audit.view", "contract.read", "customer.read", "approval.view"},                                                                                                       // 审计管理员
}

// GetRolePermissions 获取角色的默认权限列表
// 参数：role - 角色标识
// 返回：该角色拥有的权限列表，如果角色不存在则返回空数组
func GetRolePermissions(role string) []string {
	permissions, exists := RolePermissionMap[role]
	if !exists {
		return []string{}
	}
	return permissions
}

// GetAllPermissionKeys 获取所有权限的标识列表
// 返回：所有权限的key数组
func GetAllPermissionKeys() []string {
	keys := make([]string, len(AllPermissions))
	for i, p := range AllPermissions {
		keys[i] = p.Key
	}
	return keys
}

// GetPermissionsByCategory 按分组获取权限列表
// 参数：category - 权限分组名称
// 返回：该分组下的所有权限
func GetPermissionsByCategory(category string) []Permission {
	var result []Permission
	for _, p := range AllPermissions {
		if p.Category == category {
			result = append(result, p)
		}
	}
	return result
}

// GetCategories 获取所有权限分组
// 返回：所有不重复的分组名称数组
func GetCategories() []string {
	categoryMap := make(map[string]bool)
	var categories []string
	for _, p := range AllPermissions {
		if !categoryMap[p.Category] {
			categoryMap[p.Category] = true
			categories = append(categories, p.Category)
		}
	}
	return categories
}

// HasPermission 检查是否拥有指定权限
// 参数：rolePermissions - 角色权限列表, customPermissions - 用户自定义权限列表, requiredPermission - 需要检查的权限
// 返回：是否拥有该权限
func HasPermission(rolePermissions, customPermissions []string, requiredPermission string) bool {
	allPermissions := append(rolePermissions, customPermissions...)
	for _, p := range allPermissions {
		if p == "all" || p == requiredPermission {
			return true
		}
	}
	return false
}

// GetUserPermissions 获取用户的完整权限列表
// 参数：role - 用户角色, customPermissions - 用户自定义权限（JSON字符串）
// 返回：合并后的完整权限列表
func GetUserPermissions(role string, customPermissionsJSON string) []string {
	// 获取角色默认权限
	permissions := GetRolePermissions(role)

	// 解析并追加用户自定义权限
	if customPermissionsJSON != "" {
		var customPerms []string
		if err := json.Unmarshal([]byte(customPermissionsJSON), &customPerms); err == nil {
			permissions = append(permissions, customPerms...)
		}
	}

	return permissions
}

// ParseCustomPermissions 解析用户自定义权限JSON
// 参数：customPermissionsJSON - 自定义权限的JSON字符串
// 返回：权限列表和错误信息
func ParseCustomPermissions(customPermissionsJSON string) ([]string, error) {
	if customPermissionsJSON == "" {
		return []string{}, nil
	}
	var permissions []string
	err := json.Unmarshal([]byte(customPermissionsJSON), &permissions)
	return permissions, err
}

// SerializeCustomPermissions 序列化权限列表为JSON
// 参数：permissions - 权限列表
// 返回：JSON字符串
func SerializeCustomPermissions(permissions []string) string {
	data, err := json.Marshal(permissions)
	if err != nil {
		return "[]"
	}
	return string(data)
}
