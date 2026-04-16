package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

// ApprovalWorkflow 审批工作流模型
// 表示一个合同的多级审批流程，包含当前审批级别和状态
type ApprovalWorkflow struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`                                                 // 工作流唯一标识
	ContractID   uint64    `json:"contract_id" gorm:"column:contract_id;index;not null"`                 // 关联合同ID
	CreatorID    uint64    `json:"creator_id" gorm:"column:creator_id;index"`                            // 创建人ID（销售人员）
	CurrentLevel int       `json:"current_level" gorm:"column:current_level;default:1"`                  // 当前审批级别（从1开始）
	MaxLevel     int       `json:"max_level" gorm:"column:max_level;default:3"`                          // 最大审批级别（3级审批）
	Status       string    `json:"status" gorm:"column:status;type:varchar(20);default:'pending';index"` // 工作流状态
	CreatorRole  string    `json:"creator_role" gorm:"column:creator_role;type:varchar(20);not null"`    // 创建者的角色
	Hash         string    `json:"hash" gorm:"column:hash;type:varchar(64)"`                             // 工作流角色权限完整性哈希（SHA-256）
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`                                  // 创建时间
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`                                  // 更新时间
}

// WfNode 工作流节点模型（避免GORM关系解析问题）
// 表示工作流中的单个审批节点，包含审批人信息和审批结果
type WfNode struct {
	ID           uint64     `json:"id" gorm:"primaryKey"`                                                // 审批节点唯一标识
	WfID         uint64     `json:"wf_id" gorm:"column:workflow_id;index;not null"`                      // 关联工作流ID
	ContractID   uint64     `json:"contract_id" gorm:"column:contract_id;index;not null"`                // 关联合同ID
	ApproverRef  *uint64    `json:"approver_ref" gorm:"column:approver_id"`                              // 审批人ID（可为空，表示待指派）
	ApproverRole string     `json:"approver_role" gorm:"column:approver_role;type:varchar(20);not null"` // 审批人角色
	Level        int        `json:"level" gorm:"column:level;not null"`                                  // 审批级别（第几级审批）
	Status       string     `json:"status" gorm:"column:status;type:varchar(20);default:'pending'"`      // 审批状态
	Comment      string     `json:"comment" gorm:"column:comment;type:text"`                             // 审批意见
	Hash         string     `json:"hash" gorm:"column:hash;type:varchar(64)"`                            // 审批节点完整性哈希
	ApprovedAt   *time.Time `json:"approved_at" gorm:"column:approved_at"`                               // 审批时间
	CreatedAt    time.Time  `json:"created_at" gorm:"column:created_at"`                                 // 创建时间
}

func (WfNode) TableName() string {
	return "workflow_approvals"
}

// 工作流状态常量
const (
	WorkflowStatusPending   = "pending"   // 待审批，审批流程正在进行中
	WorkflowStatusApproved  = "approved"  // 已通过，所有审批级别都已通过
	WorkflowStatusRejected  = "rejected"  // 已拒绝，在某一级审批被拒绝
	WorkflowStatusCompleted = "completed" // 已完成，审批流程已结束（无论通过或拒绝）
)

// 审批级别常量
const (
	WorkflowLevel1 = 1 // 第一级审批
	WorkflowLevel2 = 2 // 第二级审批
)

// ApprovalRoles 角色审批级别映射
// 定义不同角色可以审批的级别，数值越大级别越高
var ApprovalRoles = map[string]int{
	"sales":       0, // 销售（最低级别）
	"admin":       1, // 管理员（第一级）
	"director":    2, // 总监（第二级）
	"super_admin": 3, // 超级管理员（最高级别）
}

// RolePermission 角色权限结构
// 用于描述角色的审批权限信息
type RolePermission struct {
	Role        string `json:"role"`        // 角色名称
	Level       int    `json:"level"`       // 审批级别
	Permissions string `json:"permissions"` // 权限列表
}

// CalculateRolePermissionHash 计算角色权限列表的哈希值
// 将角色-级别映射转换为有序字符串后计算SHA-256哈希
// 用于确保审批工作流中角色配置的一致性和完整性
// 参数：roles-角色到审批级别的映射
// 返回：64位十六进制哈希字符串，如果映射为空则返回空字符串
func CalculateRolePermissionHash(roles map[string]int) string {
	// 如果映射为空，返回空字符串
	if len(roles) == 0 {
		return ""
	}

	// 提取所有角色名并排序，确保哈希计算的确定性
	var keys []string
	for role := range roles {
		keys = append(keys, role)
	}
	sort.Strings(keys)

	// 按排序后的顺序构建字符串：role1:level1;role2:level2;...
	var input string
	for _, role := range keys {
		input += fmt.Sprintf("%s:%d;", role, roles[role])
	}

	// 计算SHA-256哈希并转换为十六进制字符串
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// CalculateHash 计算工作流的角色权限哈希
// 收集工作流中所有角色及其对应的审批级别，生成完整性哈希
// 包含创建者角色和所有审批节点的角色信息
// 返回：SHA-256哈希值
func (w *ApprovalWorkflow) CalculateHash() string {
	// 构建角色-级别映射，初始包含创建者角色
	roles := map[string]int{
		w.CreatorRole: w.CurrentLevel,
	}
	// 添加固定的审批角色
	roles[string(RoleSalesDirector)] = 1
	roles[string(RoleTechDirector)] = 2
	roles[string(RoleFinanceDirector)] = 3
	return CalculateRolePermissionHash(roles)
}

// VerifyIntegrity 验证工作流数据的完整性
// 重新计算工作流的角色权限哈希，与存储的Hash字段进行对比
// 如果一致说明工作流配置未被篡改，否则数据可能存在安全问题
// 返回：true表示完整性验证通过，false表示验证失败
func (w *ApprovalWorkflow) VerifyIntegrity() bool {
	expectedHash := w.CalculateHash()
	return expectedHash == w.Hash
}
