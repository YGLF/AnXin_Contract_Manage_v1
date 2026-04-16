package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"contract-manage/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// UserRole 用户角色类型
// 使用string类型定义，便于数据库存储和JSON序列化
type UserRole string

// 系统预定义的用户角色常量
// 注意：角色权限的完整性和鉴别信息已通过SHA-256哈希保护
const (
	RoleAdmin           UserRole = "admin"            // 超级管理员，拥有系统全部权限
	RoleSalesDirector   UserRole = "sales_director"   // 销售总监，审批销售提交的合同
	RoleTechDirector    UserRole = "tech_director"    // 技术总监，审批技术相关合同
	RoleFinanceDirector UserRole = "finance_director" // 财务总监，审批财务相关合同
	RoleContractAdmin   UserRole = "contract_admin"   // 合同管理员，可查看所有合同
	RoleSales           UserRole = "sales"            // 销售人员，只能查看自己的合同
	RoleAuditAdmin      UserRole = "audit_admin"      // 审计管理员，负责系统操作日志审计
)

// 角色中文名称映射
var RoleDisplayNames = map[UserRole]string{
	RoleAdmin:           "超级管理员",
	RoleSalesDirector:   "销售总监",
	RoleTechDirector:    "技术总监",
	RoleFinanceDirector: "财务总监",
	RoleContractAdmin:   "合同管理员",
	RoleSales:           "销售人员",
	RoleAuditAdmin:      "审计管理员",
}

// GetRoleDisplayName 获取角色的中文显示名称
func GetRoleDisplayName(role UserRole) string {
	if name, ok := RoleDisplayNames[role]; ok {
		return name
	}
	return string(role)
}

// IsDirectorRole 判断是否为总监角色
func IsDirectorRole(role UserRole) bool {
	return role == RoleSalesDirector || role == RoleTechDirector || role == RoleFinanceDirector
}

// IsApprovalRole 判断是否为审批角色（总监及以上）
func IsApprovalRole(role UserRole) bool {
	return IsDirectorRole(role) || role == RoleAdmin
}

// ContractStatus 合同状态类型
// 用于标识合同在整个生命周期中所处的阶段
type ContractStatus string

// 合同状态常量定义
// 状态流转：draft -> pending -> approved -> active -> in_progress -> pending_pay -> completed -> archived
const (
	StatusDraft      ContractStatus = "draft"       // 草稿状态，合同正在编辑，尚未提交
	StatusPending    ContractStatus = "pending"     // 待审批状态，已提交等待审批
	StatusApproved   ContractStatus = "approved"    // 已批准状态，审批通过等待生效
	StatusActive     ContractStatus = "active"      // 已生效状态，合同已生效可以执行
	StatusInProgress ContractStatus = "in_progress" // 执行中状态，合同正在执行中
	StatusPendingPay ContractStatus = "pending_pay" // 待付款状态，等待付款完成
	StatusCompleted  ContractStatus = "completed"   // 已完成状态，合同执行完毕
	StatusTerminated ContractStatus = "terminated"  // 已终止状态，合同提前终止
	StatusArchived   ContractStatus = "archived"    // 已归档状态，合同已归档保存
)

// 合同状态的中文显示文本
const (
	StatusDraftText      = "草稿"
	StatusPendingText    = "待审批"
	StatusApprovedText   = "已批准"
	StatusActiveText     = "已生效"
	StatusInProgressText = "执行中"
	StatusPendingPayText = "待付款"
	StatusCompletedText  = "已完成"
	StatusTerminatedText = "已终止"
	StatusArchivedText   = "已归档"
)

// GetStatusText 获取合同状态的中文描述
// 参数：status 合同状态枚举值
// 返回：状态对应的中文描述文本
func GetStatusText(status ContractStatus) string {
	switch status {
	case StatusDraft:
		return StatusDraftText
	case StatusPending:
		return StatusPendingText
	case StatusApproved:
		return StatusApprovedText
	case StatusActive:
		return StatusActiveText
	case StatusInProgress:
		return StatusInProgressText
	case StatusPendingPay:
		return StatusPendingPayText
	case StatusCompleted:
		return StatusCompletedText
	case StatusTerminated:
		return StatusTerminatedText
	case StatusArchived:
		return StatusArchivedText
	default:
		return string(status)
	}
}

// GetStatusOptions 获取合同状态选项列表
// 返回：包含所有状态选项的切片，每个元素包含value和label
// 用于前端下拉选择框等场景
func GetStatusOptions() []map[string]string {
	return []map[string]string{
		{"value": string(StatusDraft), "label": StatusDraftText},
		{"value": string(StatusPending), "label": StatusPendingText},
		{"value": string(StatusApproved), "label": StatusApprovedText},
		{"value": string(StatusActive), "label": StatusActiveText},
		{"value": string(StatusInProgress), "label": StatusInProgressText},
		{"value": string(StatusPendingPay), "label": StatusPendingPayText},
		{"value": string(StatusCompleted), "label": StatusCompletedText},
		{"value": string(StatusTerminated), "label": StatusTerminatedText},
		{"value": string(StatusArchived), "label": StatusArchivedText},
	}
}

// ApprovalStatus 审批状态类型
// 用于标识审批记录的处理结果
type ApprovalStatus string

// 审批状态常量定义
const (
	ApprovalPending  ApprovalStatus = "pending"  // 待审批，审批人尚未处理
	ApprovalApproved ApprovalStatus = "approved" // 已批准，审批人同意申请
	ApprovalRejected ApprovalStatus = "rejected" // 已拒绝，审批人拒绝申请
)

// UserStatusType 用户账号状态类型
// 用于标识用户账号的有效期类型
type UserStatusType string

// 用户账号状态常量定义
const (
	UserStatusPermanent UserStatusType = "permanent" // 长期有效，账号永久有效
	UserStatusTemporary UserStatusType = "temporary" // 临时账号，短期内有效
	UserStatusDisabled  UserStatusType = "disabled"  // 禁用状态，账号被禁用
	UserStatusTimed     UserStatusType = "timed"     // 指定时间段，在有效期内可用
)

// User 用户模型
// 存储系统用户信息，包含身份鉴别信息和完整性保护哈希值
type User struct {
	ID                uint             `gorm:"primaryKey" json:"id"`                                    // 用户唯一标识
	Username          string           `gorm:"size:50;uniqueIndex;not null" json:"username"`            // 用户名，唯一索引
	Email             string           `gorm:"size:100;index" json:"email"`                             // 邮箱，可重复
	HashedPassword    string           `gorm:"size:200;not null" json:"-"`                              // bcrypt加密后的密码，不返回给前端
	PasswordHash      string           `gorm:"size:64" json:"password_hash"`                            // 密码SHA-256杂凑值，用于登录验证
	HashVerified      bool             `gorm:"default:false" json:"hash_verified"`                      // 杂凑验证是否通过
	IntegrityHash     string           `gorm:"size:64" json:"integrity_hash"`                           // 用户鉴别信息完整性哈希（SHA-256）
	FullName          string           `gorm:"size:100" json:"full_name"`                               // 真实姓名
	Role              UserRole         `gorm:"size:20;default:user" json:"role"`                        // 用户角色
	CustomPermissions string           `gorm:"type:text" json:"custom_permissions"`                     // 用户自定义权限（JSON数组格式）
	Department        string           `gorm:"size:100" json:"department"`                              // 所属部门
	Phone             string           `gorm:"size:20" json:"phone"`                                    // 联系电话
	IsActive          bool             `gorm:"default:true" json:"is_active"`                           // 是否激活（旧字段，保留兼容）
	AccountStatus     UserStatusType   `gorm:"size:20;default:permanent" json:"account_status"`         // 账号状态类型：permanent/temporary/disabled/timed
	ValidFrom         *time.Time       `gorm:"type:datetime" json:"valid_from"`                         // 账号有效期开始时间
	ValidTo           *time.Time       `gorm:"type:datetime" json:"valid_to"`                           // 账号有效期结束时间
	ValidHours        int              `gorm:"default:0" json:"valid_hours"`                            // 临时账号有效期小时数
	CreatedAt         time.Time        `json:"created_at"`                                              // 创建时间
	UpdatedAt         *time.Time       `json:"updated_at"`                                              // 更新时间
	IsLocked          bool             `gorm:"-" json:"is_locked"`                                      // 账号是否被锁定（用于显示，不存储到数据库）
	Contracts         []Contract       `gorm:"foreignKey:CreatorID" json:"contracts,omitempty"`         // 创建的合同
	ApprovalRecords   []ApprovalRecord `gorm:"foreignKey:ApproverID" json:"approval_records,omitempty"` // 审批记录
}

// IsAccountValid 检查用户账号是否在有效期内
// 根据账号状态类型和有效期时间段判断账号是否可用
// 返回：账号是否有效
func (u *User) IsAccountValid() bool {
	// 如果账号状态为禁用，直接返回false
	if u.AccountStatus == UserStatusDisabled {
		return false
	}

	// 如果账号状态为永久有效，返回true
	if u.AccountStatus == UserStatusPermanent {
		return true
	}

	// 如果账号状态为临时有效，检查有效期小时数
	if u.AccountStatus == UserStatusTemporary {
		now := time.Now()

		// 如果设置了有效小时数
		if u.ValidHours > 0 && u.ValidFrom != nil {
			expireTime := u.ValidFrom.Add(time.Duration(u.ValidHours) * time.Hour)
			if now.After(expireTime) {
				return false
			}
			return true
		}

		// 如果有结束时间，检查是否已过期
		if u.ValidTo != nil {
			if now.After(*u.ValidTo) {
				return false
			}
			return true
		}

		return false
	}

	// 如果账号状态为指定时间段
	if u.AccountStatus == UserStatusTimed {
		now := time.Now()

		// 如果有开始时间，检查是否已生效
		if u.ValidFrom != nil && now.Before(*u.ValidFrom) {
			return false
		}

		// 如果有结束时间，检查是否已过期
		if u.ValidTo != nil && now.After(*u.ValidTo) {
			return false
		}

		return true
	}

	// 默认返回is_active字段的值
	return u.IsActive
}

// GetAccountStatusText 获取账号状态的中文描述
// 返回：账号状态对应的中文文本
func (u *User) GetAccountStatusText() string {
	switch u.AccountStatus {
	case UserStatusPermanent:
		return "长期"
	case UserStatusTemporary:
		if u.ValidHours > 0 {
			return "临时"
		}
		return "临时"
	case UserStatusDisabled:
		return "禁用"
	case UserStatusTimed:
		return "指定时间段"
	default:
		return "长期"
	}
}

// GetRemainingHours 获取临时账号剩余有效小时数
// 返回：剩余小时数，如果账号已过期返回0
func (u *User) GetRemainingHours() int {
	if u.AccountStatus != UserStatusTemporary {
		return 0
	}

	if u.ValidHours <= 0 || u.ValidFrom == nil {
		return 0
	}

	now := time.Now()
	expireTime := u.ValidFrom.Add(time.Duration(u.ValidHours) * time.Hour)
	remaining := expireTime.Sub(now).Hours()

	if remaining < 0 {
		return 0
	}

	return int(remaining)
}

// Role 角色模型
// 存储角色定义及其权限信息，包含权限完整性哈希
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                // 角色唯一标识
	Name        string    `gorm:"size:50;unique;not null" json:"name"` // 角色名称，唯一
	Description string    `gorm:"type:text" json:"description"`        // 角色描述
	Permissions string    `gorm:"type:text" json:"permissions"`        // 权限列表，逗号分隔
	Hash        string    `gorm:"size:64" json:"hash"`                 // 角色权限完整性哈希（SHA-256）
	CreatedAt   time.Time `json:"created_at"`                          // 创建时间
}

// Customer 客户/供应商模型
// 存储客户或供应商的基本信息
type Customer struct {
	ID            uint       `gorm:"primaryKey" json:"id"`                             // 客户唯一标识
	Name          string     `gorm:"size:200;not null;index" json:"name"`              // 客户名称，必填
	Type          string     `gorm:"size:20;default:customer" json:"type"`             // 客户类型：customer=客户，supplier=供应商
	Code          string     `gorm:"size:50;uniqueIndex" json:"code"`                  // 客户编码，唯一索引
	ContactPerson string     `gorm:"size:100" json:"contact_person"`                   // 联系人
	ContactPhone  string     `gorm:"size:20" json:"contact_phone"`                     // 联系电话
	ContactEmail  string     `gorm:"size:100" json:"contact_email"`                    // 联系邮箱
	Address       string     `gorm:"type:text" json:"address"`                         // 地址
	CreditRating  string     `gorm:"size:20" json:"credit_rating"`                     // 信用等级
	IsActive      bool       `gorm:"default:true" json:"is_active"`                    // 是否启用
	CreatedAt     time.Time  `json:"created_at"`                                       // 创建时间
	UpdatedAt     *time.Time `json:"updated_at"`                                       // 更新时间
	Contracts     []Contract `gorm:"foreignKey:CustomerID" json:"contracts,omitempty"` // 关联的合同
}

// ContractType 合同类型模型
// 用于对合同进行分类管理
type ContractType struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                 // 类型唯一标识
	Name        string    `gorm:"size:100;unique;not null" json:"name"` // 类型名称，唯一
	Code        string    `gorm:"size:50;unique" json:"code"`           // 类型编码，唯一
	Description string    `gorm:"type:text" json:"description"`         // 类型描述
	CreatedAt   time.Time `json:"created_at"`                           // 创建时间
}

// Contract 合同模型
// 存储合同的核心信息，关联客户、类型、创建人等
type Contract struct {
	ID              uint                `gorm:"primaryKey" json:"id"`                                     // 合同唯一标识
	ContractNo      string              `gorm:"size:50;uniqueIndex;not null" json:"contract_no"`          // 合同编号，唯一索引
	Title           string              `gorm:"size:200;not null;index" json:"title"`                     // 合同标题
	CustomerID      uint                `gorm:"index" json:"customer_id"`                                 // 关联客户ID
	ContractTypeID  uint                `gorm:"index" json:"contract_type_id"`                            // 关联合同类型ID
	Amount          float64             `json:"amount"`                                                   // 合同金额
	Currency        string              `gorm:"size:10;default:CNY" json:"currency"`                      // 货币单位，默认人民币
	Status          ContractStatus      `gorm:"size:20;default:draft" json:"status"`                      // 合同状态
	SignDate        *time.Time          `json:"sign_date"`                                                // 签订日期
	StartDate       *time.Time          `json:"start_date"`                                               // 开始日期
	EndDate         *time.Time          `json:"end_date"`                                                 // 结束日期
	PaymentTerms    string              `gorm:"type:text" json:"payment_terms"`                           // 付款条款
	Content         string              `gorm:"type:text" json:"content"`                                 // 合同正文内容
	Notes           string              `gorm:"type:text" json:"notes"`                                   // 备注
	CreatorID       uint                `gorm:"index" json:"creator_id"`                                  // 创建人ID
	CreatedAt       time.Time           `json:"created_at"`                                               // 创建时间
	UpdatedAt       *time.Time          `json:"updated_at"`                                               // 更新时间
	Customer        *Customer           `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`          // 关联客户
	Creator         *User               `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`            // 创建人
	ContractType    *ContractType       `gorm:"foreignKey:ContractTypeID" json:"contract_type,omitempty"` // 合同类型
	Executions      []ContractExecution `gorm:"foreignKey:ContractID" json:"executions,omitempty"`        // 执行记录
	Documents       []Document          `gorm:"foreignKey:ContractID" json:"documents,omitempty"`         // 合同文档
	ApprovalRecords []ApprovalRecord    `gorm:"foreignKey:ContractID" json:"approval_records,omitempty"`  // 审批记录
	Reminders       []Reminder          `gorm:"foreignKey:ContractID" json:"reminders,omitempty"`         // 提醒记录
}

// ContractExecution 合同执行记录模型
// 记录合同的执行阶段、进度和付款信息
type ContractExecution struct {
	ID            uint       `gorm:"primaryKey" json:"id"`                            // 执行记录唯一标识
	ContractID    uint       `gorm:"index" json:"contract_id"`                        // 关联合同ID
	Stage         string     `gorm:"size:100" json:"stage"`                           // 执行阶段名称
	StageDate     *time.Time `json:"stage_date"`                                      // 阶段日期
	Progress      float64    `gorm:"default:0" json:"progress"`                       // 进度百分比（0-100）
	PaymentAmount float64    `json:"payment_amount"`                                  // 付款金额
	PaymentDate   *time.Time `json:"payment_date"`                                    // 付款日期
	Description   string     `gorm:"type:text" json:"description"`                    // 执行描述
	OperatorID    uint       `gorm:"index" json:"operator_id"`                        // 操作人ID
	CreatedAt     time.Time  `json:"created_at"`                                      // 创建时间
	Contract      *Contract  `gorm:"foreignKey:ContractID" json:"contract,omitempty"` // 关联合同
}

// ApprovalRecord 审批记录模型
// 存储合同的审批历史和审批人意见
type ApprovalRecord struct {
	ID           uint           `gorm:"primaryKey" json:"id"`                            // 审批记录唯一标识
	ContractID   uint           `gorm:"index" json:"contract_id"`                        // 关联合同ID
	ApproverID   uint           `gorm:"index" json:"approver_id"`                        // 审批人ID
	Level        int            `gorm:"default:1" json:"level"`                          // 审批级别（第几级审批）
	ApproverRole string         `gorm:"size:20" json:"approver_role"`                    // 审批人角色
	Status       ApprovalStatus `gorm:"size:20;default:pending" json:"status"`           // 审批状态
	Comment      string         `gorm:"type:text" json:"comment"`                        // 审批意见
	ApprovedAt   *time.Time     `json:"approved_at"`                                     // 审批时间
	CreatedAt    time.Time      `json:"created_at"`                                      // 创建时间
	DueAt        *time.Time     `json:"due_at"`                                          // 审批截止时间（超时时间）
	IsExpired    bool           `gorm:"default:false" json:"is_expired"`                 // 是否已过期
	Contract     *Contract      `gorm:"foreignKey:ContractID" json:"contract,omitempty"` // 关联合同
	Approver     *User          `gorm:"foreignKey:ApproverID" json:"approver,omitempty"` // 审批人
}

// IsApprovalExpired 检查审批是否已超时
func (a *ApprovalRecord) IsApprovalExpired() bool {
	if a.IsExpired {
		return true
	}
	if a.DueAt != nil && time.Now().After(*a.DueAt) {
		return true
	}
	return false
}

// GetApprovalTimeout 检查审批超时配置
// 默认超时时间为72小时（3天）
const DefaultApprovalTimeoutHours = 72

// Document 合同文档模型
// 存储合同相关的附件文档信息
type Document struct {
	ID         uint      `gorm:"primaryKey" json:"id"`                            // 文档唯一标识
	ContractID uint      `gorm:"index" json:"contract_id"`                        // 关联合同ID
	Name       string    `gorm:"size:200" json:"name"`                            // 文档名称
	FilePath   string    `gorm:"size:500" json:"file_path"`                       // 文件存储路径
	FileSize   int       `json:"file_size"`                                       // 文件大小（字节）
	FileType   string    `gorm:"size:50" json:"file_type"`                        // 文件类型（如pdf、docx）
	Version    string    `gorm:"size:20;default:1.0" json:"version"`              // 文档版本
	UploaderID uint      `gorm:"index" json:"uploader_id"`                        // 上传人ID
	CreatedAt  time.Time `json:"created_at"`                                      // 上传时间
	Contract   *Contract `gorm:"foreignKey:ContractID" json:"contract,omitempty"` // 关联合同
}

// LifecycleEventType 合同生命周期事件类型
// 用于记录合同状态变更的历史
type LifecycleEventType string

// 生命周期事件类型常量
const (
	LifecycleCreated    LifecycleEventType = "created"    // 合同创建
	LifecycleSubmitted  LifecycleEventType = "submitted"  // 提交审批
	LifecycleApproved   LifecycleEventType = "approved"   // 审批通过
	LifecycleRejected   LifecycleEventType = "rejected"   // 审批拒绝
	LifecycleActivated  LifecycleEventType = "activated"  // 合同生效
	LifecycleProgress   LifecycleEventType = "progress"   // 执行进度更新
	LifecyclePayment    LifecycleEventType = "payment"    // 付款记录
	LifecycleCompleted  LifecycleEventType = "completed"  // 合同完成
	LifecycleTerminated LifecycleEventType = "terminated" // 合同终止
	LifecycleArchived   LifecycleEventType = "archived"   // 合同归档
)

// ContractLifecycleEvent 合同生命周期事件模型
// 记录合同整个生命周期中的所有重要事件
type ContractLifecycleEvent struct {
	ID          uint               `gorm:"primaryKey" json:"id"`                            // 事件唯一标识
	ContractID  uint               `gorm:"index" json:"contract_id"`                        // 关联合同ID
	EventType   LifecycleEventType `gorm:"size:50" json:"event_type"`                       // 事件类型
	FromStatus  string             `gorm:"size:50" json:"from_status"`                      // 变更前状态
	ToStatus    string             `gorm:"size:50" json:"to_status"`                        // 变更后状态
	Amount      float64            `json:"amount"`                                          // 涉及金额（如付款金额）
	Description string             `gorm:"type:text" json:"description"`                    // 事件描述
	OperatorID  uint               `gorm:"index" json:"operator_id"`                        // 操作人ID
	CreatedAt   time.Time          `json:"created_at"`                                      // 事件发生时间
	Contract    *Contract          `gorm:"foreignKey:ContractID" json:"contract,omitempty"` // 关联合同
}

// StatusChangeRequest 合同状态变更申请模型
// 用于需要审批的状态变更（如归档、终止等）
type StatusChangeRequest struct {
	ID          uint       `gorm:"primaryKey" json:"id"`                              // 申请唯一标识
	ContractID  uint       `gorm:"index" json:"contract_id"`                          // 关联合同ID
	FromStatus  string     `gorm:"size:50" json:"from_status"`                        // 原状态
	ToStatus    string     `gorm:"size:50" json:"to_status"`                          // 目标状态
	Reason      string     `gorm:"type:text" json:"reason"`                           // 变更原因
	RequesterID uint       `gorm:"index" json:"requester_id"`                         // 申请人ID
	ApproverID  *uint      `gorm:"index" json:"approver_id,omitempty"`                // 审批人ID
	Status      string     `gorm:"size:20;default:pending" json:"status"`             // 申请状态：pending/approved/rejected
	Comment     string     `gorm:"type:text" json:"comment"`                          // 审批意见
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`                             // 审批时间
	CreatedAt   time.Time  `json:"created_at"`                                        // 申请时间
	UpdatedAt   time.Time  `json:"updated_at"`                                        // 更新时间
	Contract    *Contract  `gorm:"foreignKey:ContractID" json:"contract,omitempty"`   // 关联合同
	Requester   *User      `gorm:"foreignKey:RequesterID" json:"requester,omitempty"` // 申请人
	Approver    *User      `gorm:"foreignKey:ApproverID" json:"approver,omitempty"`   // 审批人
}

// Reminder 合同到期提醒模型
// 用于设置合同到期前的提醒通知
type Reminder struct {
	ID           uint       `gorm:"primaryKey" json:"id"`         // 提醒唯一标识
	ContractID   uint       `gorm:"index" json:"contract_id"`     // 关联合同ID
	Type         string     `gorm:"size:50" json:"type"`          // 提醒类型（如expire_payment-付款到期, expire_end-合同到期）
	ReminderDate *time.Time `json:"reminder_date"`                // 提醒日期
	DaysBefore   int        `json:"days_before"`                  // 提前提醒天数
	IsSent       bool       `gorm:"default:false" json:"is_sent"` // 是否已发送
	SentAt       *time.Time `json:"sent_at"`                      // 发送时间
	CreatedAt    time.Time  `json:"created_at"`                   // 创建时间
}

// Notification 审批提醒通知模型
// 用于向用户发送审批提醒通知
type Notification struct {
	ID         uint       `gorm:"primaryKey" json:"id"`                         // 通知唯一标识
	UserID     uint       `gorm:"index" json:"user_id"`                         // 接收用户ID
	ContractID uint       `gorm:"index" json:"contract_id"`                     // 关联合同ID
	WorkflowID uint       `gorm:"index" json:"workflow_id"`                     // 关联工作流ID
	Role       string     `gorm:"column:target_role;size:20" json:"role"`       // 目标角色
	Type       string     `gorm:"column:notification_type;size:50" json:"type"` // 通知类型
	Title      string     `gorm:"size:200" json:"title"`                        // 通知标题
	Content    string     `gorm:"column:content;type:text" json:"content"`      // 通知内容
	IsRead     bool       `gorm:"column:is_read;default:false" json:"is_read"`  // 是否已读
	ReadAt     *time.Time `json:"read_at"`                                      // 阅读时间
	CreatedAt  time.Time  `json:"created_at"`                                   // 创建时间
}

// NotificationType 通知类型常量
const (
	NotificationTypeApprovalReminder = "approval_reminder" // 审批提醒
	NotificationTypeApproved         = "approved"          // 审批通过
	NotificationTypeRejected         = "rejected"          // 审批拒绝
	NotificationTypePendingApproval  = "pending_approval"  // 待审批通知
	NotificationTypeStatusChange     = "status_change"     // 状态变更通知
)

// AuditLog 审计日志模型
// 记录用户在系统中的所有操作行为，用于安全审计
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`                    // 日志唯一标识
	UserID     uint      `gorm:"index" json:"user_id"`                    // 操作用户ID
	Username   string    `gorm:"size:100" json:"username"`                // 操作用户名
	Action     string    `gorm:"size:100" json:"action"`                  // 操作动作（如create, update, delete）
	Module     string    `gorm:"size:50" json:"module"`                   // 操作模块（如customer, contract）
	Method     string    `gorm:"size:20" json:"method"`                   // HTTP请求方法
	Path       string    `gorm:"size:255" json:"path"`                    // 请求路径
	IPAddress  string    `gorm:"size:50" json:"ip_address"`               // 客户端IP地址
	UserAgent  string    `gorm:"type:text" json:"user_agent"`             // 客户端浏览器信息
	Request    string    `gorm:"type:text" json:"request"`                // 请求内容
	Response   string    `gorm:"type:text" json:"response"`               // 响应内容
	StatusCode int       `json:"status_code"`                             // HTTP响应状态码
	CreatedAt  time.Time `json:"created_at"`                              // 操作时间
	User       *User     `gorm:"foreignKey:UserID" json:"user,omitempty"` // 操作用户
}

// LoginFailureRecord 登录失败记录模型
// 用于追踪用户登录失败次数，防止暴力破解
type LoginFailureRecord struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Username    string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	FailCount   int        `gorm:"default:1" json:"fail_count"`
	FirstFail   time.Time  `gorm:"not null" json:"first_fail"`
	LastFail    time.Time  `gorm:"not null" json:"last_fail"`
	Locked      bool       `gorm:"default:false" json:"locked"`
	LockedUntil *time.Time `gorm:"type:datetime" json:"locked_until"`
}

// 登录安全配置常量
const (
	MaxLoginAttempts    = 5                // 最大登录尝试次数
	LockoutDuration     = 15 * time.Minute // 锁定时长（分钟）
	FailureResetMinutes = 30               // 失败记录重置时间（分钟）
)

// DB 全局数据库连接实例
// 所有数据库操作都通过此实例进行
var DB *gorm.DB

// InitDB 初始化数据库连接
// 根据配置文件中的数据库参数建立MySQL连接，并执行自动迁移
// 返回：错误信息，如果连接成功则返回nil
func InitDB() error {
	// 构建数据库连接字符串（Data Source Name）
	// 格式：用户名:密码@tcp(主机:端口)/数据库名?字符集设置
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.MysqlUser,     // 数据库用户名
		config.AppConfig.MysqlPassword, // 数据库密码
		config.AppConfig.MysqlHost,     // 数据库主机地址
		config.AppConfig.MysqlPort,     // 数据库端口号
		config.AppConfig.MysqlDatabase, // 数据库名称
	)

	// 使用GORM打开MySQL连接
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err // 返回连接错误
	}

	// 执行数据库表自动迁移
	return AutoMigrate()
}

// AutoMigrate 自动迁移数据库表结构
// 根据模型定义自动创建或更新表结构，确保数据库表与模型同步
// 会创建所有缺失的表和字段，但不会删除已有数据
func AutoMigrate() error {
	err := DB.AutoMigrate(
		&User{},                   // 用户表
		&Role{},                   // 角色表
		&Customer{},               // 客户/供应商表
		&ContractType{},           // 合同类型表
		&Contract{},               // 合同表
		&ContractExecution{},      // 合同执行记录表
		&ApprovalRecord{},         // 审批记录表
		&Document{},               // 文档表
		&ContractLifecycleEvent{}, // 合同生命周期事件表
		&Reminder{},               // 提醒表
		&Notification{},           // 通知表
		&StatusChangeRequest{},    // 状态变更申请表
		&AuditLog{},               // 审计日志表
		&LoginFailureRecord{},     // 登录失败记录表
		&ApprovalWorkflow{},       // 审批工作流表
		&WfNode{},                 // 审批节点表
	)
	if err != nil {
		return err
	}

	// 确保用户表有新字段（兼容旧数据库）
	if err := migrateUserFields(); err != nil {
		// 记录错误但继续运行，因为字段可能已存在
		fmt.Printf("用户表字段迁移信息: %v\n", err)
	}

	// 确保工作流表已创建（手动创建以防AutoMigrate失败）
	if err := migrateWorkflowTables(); err != nil {
		fmt.Printf("工作流表迁移信息: %v\n", err)
	}

	// 修复 email 字段索引问题（唯一索引改为普通索引）
	if err := migrateUserEmailIndex(); err != nil {
		fmt.Printf("用户表索引迁移信息: %v\n", err)
	}

	return nil
}

// migrateWorkflowTables 确保工作流表存在
func migrateWorkflowTables() error {
	// 检查 approval_workflows 表是否存在
	var count1 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'approval_workflows'").Scan(&count1)
	if count1 == 0 {
		fmt.Println("创建 approval_workflows 表...")
		DB.Exec(`
			CREATE TABLE approval_workflows (
				id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				contract_id BIGINT UNSIGNED NOT NULL,
				creator_id BIGINT UNSIGNED,
				current_level INT DEFAULT 1,
				max_level INT DEFAULT 3,
				status VARCHAR(20) DEFAULT 'pending',
				creator_role VARCHAR(20) NOT NULL,
				hash VARCHAR(64),
				created_at DATETIME,
				updated_at DATETIME,
				INDEX idx_contract_id (contract_id),
				INDEX idx_creator_id (creator_id),
				INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
		`)
	}

	// 检查 workflow_approvals 表是否存在
	var count2 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'workflow_approvals'").Scan(&count2)
	if count2 == 0 {
		fmt.Println("创建 workflow_approvals 表...")
		DB.Exec(`
			CREATE TABLE workflow_approvals (
				id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				workflow_id BIGINT UNSIGNED NOT NULL,
				contract_id BIGINT UNSIGNED NOT NULL,
				approver_id BIGINT UNSIGNED,
				approver_role VARCHAR(20) NOT NULL,
				level INT NOT NULL,
				status VARCHAR(20) DEFAULT 'pending',
				comment TEXT,
				hash VARCHAR(64),
				approved_at DATETIME,
				created_at DATETIME,
				INDEX idx_workflow_id (workflow_id),
				INDEX idx_contract_id (contract_id),
				INDEX idx_approver_role (approver_role),
				INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
		`)
	}

	return nil
}

// migrateUserFields 确保用户表有新字段
// 使用原始SQL确保字段存在，兼容已有数据库
func migrateUserFields() error {
	// 检查并添加 account_status 字段
	var count1 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'account_status'").Scan(&count1)
	if count1 == 0 {
		fmt.Println("添加 account_status 字段...")
		DB.Exec("ALTER TABLE users ADD COLUMN account_status VARCHAR(20) DEFAULT 'permanent' AFTER is_active")
	}

	// 检查并添加 valid_from 字段
	var count2 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'valid_from'").Scan(&count2)
	if count2 == 0 {
		fmt.Println("添加 valid_from 字段...")
		DB.Exec("ALTER TABLE users ADD COLUMN valid_from DATETIME AFTER account_status")
	}

	// 检查并添加 valid_to 字段
	var count3 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'valid_to'").Scan(&count3)
	if count3 == 0 {
		fmt.Println("添加 valid_to 字段...")
		DB.Exec("ALTER TABLE users ADD COLUMN valid_to DATETIME AFTER valid_from")
	}

	// 检查并添加 valid_hours 字段
	var count4 int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'valid_hours'").Scan(&count4)
	if count4 == 0 {
		fmt.Println("添加 valid_hours 字段...")
		DB.Exec("ALTER TABLE users ADD COLUMN valid_hours INT DEFAULT 0 AFTER valid_to")
	}

	return nil
}

// InitAdmin 初始化系统管理员账户
// 检查数据库中是否已存在管理员账号，如果不存在则创建
// 管理员账号信息从配置文件中读取，包括用户名、邮箱、密码
// 密码使用bcrypt加密存储，并生成用户鉴别信息完整性哈希
func InitAdmin() error {
	// 查询是否已存在超级管理员
	var existingUser User
	err := DB.Where("username = ?", config.AppConfig.AdminUsername).First(&existingUser).Error

	// 如果已存在，跳过创建
	if err == nil {
		fmt.Printf("管理员 %s 已存在\n", config.AppConfig.AdminUsername)
	} else {
		// 如果不存在，则创建管理员账号
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		// 使用bcrypt加密密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(config.AppConfig.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// 计算密码的SHA-256杂凑值
		passwordHash := CalculatePasswordHash(config.AppConfig.AdminPassword)

		// 创建管理员用户，生成完整性哈希
		admin := User{
			Username:       config.AppConfig.AdminUsername,
			Email:          config.AppConfig.AdminEmail,
			HashedPassword: string(hashedPassword),
			PasswordHash:   passwordHash,
			HashVerified:   true,
			IntegrityHash:  CalculateUserIntegrityHash(config.AppConfig.AdminUsername, config.AppConfig.AdminEmail, string(hashedPassword)),
			FullName:       "系统管理员",
			Role:           RoleAdmin,
			IsActive:       true,
		}

		if err := DB.Create(&admin).Error; err != nil {
			return err
		}
		fmt.Printf("超级管理员已创建: %s\n", config.AppConfig.AdminUsername)
	}

	// 检查并创建审计管理员账号
	var existingAuditAdmin User
	err = DB.Where("username = ?", config.AppConfig.AuditAdminUsername).First(&existingAuditAdmin).Error

	if err == nil {
		fmt.Printf("审计管理员 %s 已存在\n", config.AppConfig.AuditAdminUsername)
		return nil
	}

	// 创建审计管理员账号
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(config.AppConfig.AuditAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	passwordHash := CalculatePasswordHash(config.AppConfig.AuditAdminPassword)

	auditAdmin := User{
		Username:       config.AppConfig.AuditAdminUsername,
		Email:          config.AppConfig.AuditAdminEmail,
		HashedPassword: string(hashedPassword),
		PasswordHash:   passwordHash,
		HashVerified:   true,
		IntegrityHash:  CalculateUserIntegrityHash(config.AppConfig.AuditAdminUsername, config.AppConfig.AuditAdminEmail, string(hashedPassword)),
		FullName:       "审计管理员",
		Role:           RoleAuditAdmin,
		IsActive:       true,
		AccountStatus:  UserStatusPermanent,
	}
	if err := DB.Create(&auditAdmin).Error; err != nil {
		return err
	}
	fmt.Printf("创建审计管理员账号: %s\n", config.AppConfig.AuditAdminUsername)
	return nil
}

// migrateUserEmailIndex 修复 email 字段索引问题
// 将 email 的唯一索引改为普通索引，允许 email 为空或重复
func migrateUserEmailIndex() error {
	var count int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND INDEX_NAME = 'idx_users_email' AND NON_UNIQUE = 0").Scan(&count)
	if count > 0 {
		fmt.Println("修复 users 表 email 索引: 从唯一索引改为普通索引...")
		DB.Exec("ALTER TABLE users DROP INDEX idx_users_email")
		DB.Exec("ALTER TABLE users ADD INDEX idx_users_email (email)")
	}
	return nil
}

// CalculateRoleHash 计算角色权限哈希值
// 使用SHA-256算法对角色名称和权限字符串进行哈希
// 用于确保角色权限数据的完整性
// 参数：name-角色名称，permissions-权限列表（逗号分隔）
// 返回：64位十六进制哈希字符串
func CalculateRoleHash(name, permissions string) string {
	data := fmt.Sprintf("%s:%s", name, permissions)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CalculateUserIntegrityHash 计算用户鉴别信息完整性哈希
// 使用SHA-256算法对用户的鉴别信息进行哈希
// 将用户名、邮箱、加密后的密码组合后计算哈希
// 用于检测用户鉴别信息是否被篡改
// 参数：username-用户名，email-邮箱，hashedPassword-bcrypt加密后的密码
// 返回：64位十六进制哈希字符串
func CalculateUserIntegrityHash(username, email, hashedPassword string) string {
	data := fmt.Sprintf("%s:%s:%s", username, email, hashedPassword)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CalculatePasswordHash 计算密码的SHA-256杂凑值
// 用于登录时前端传输的密码杂凑值与数据库存储的杂凑值进行比对
// 参数：password-用户输入的明文密码
// 返回：64位十六进制杂凑字符串
func CalculatePasswordHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyIntegrity 验证用户鉴别信息完整性
// 重新计算用户鉴别信息的哈希值，并与存储的IntegrityHash进行对比
// 如果一致说明数据未被篡改，否则可能存在安全问题
// 返回：true表示完整性验证通过，false表示验证失败
func (u *User) VerifyIntegrity() bool {
	expectedHash := CalculateUserIntegrityHash(u.Username, u.Email, u.HashedPassword)
	return expectedHash == u.IntegrityHash
}

// InitRoles 初始化系统预定义角色
// 创建系统默认的角色定义，包括超级管理员、经理、销售、审计管理员
// 每个角色都会生成权限哈希值用于完整性保护
// 如果角色已存在但权限发生变化，会自动更新
func InitRoles() error {
	// 定义系统预置角色
	roles := []Role{
		{Name: "admin", Description: "超级管理员", Permissions: "all"},
		{Name: "manager", Description: "经理", Permissions: "read,write,approve"},
		{Name: "user", Description: "销售", Permissions: "read"},
		{Name: "audit_admin", Description: "审计管理员", Permissions: "audit,read"},
	}

	// 为每个角色计算权限哈希
	for i := range roles {
		roles[i].Hash = CalculateRoleHash(roles[i].Name, roles[i].Permissions)
	}

	// 遍历创建或更新角色
	for _, role := range roles {
		var existing Role
		err := DB.Where("name = ?", role.Name).First(&existing).Error
		if err == nil {
			// 角色已存在，检查是否需要更新
			if existing.Hash != role.Hash {
				existing.Hash = role.Hash
				existing.Description = role.Description
				existing.Permissions = role.Permissions
				DB.Save(&existing)
			}
			continue
		}
		// 如果记录不存在，则创建新角色
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := DB.Create(&role).Error; err != nil {
			return err
		}
		fmt.Printf("角色已创建: %s (Hash: %s)\n", role.Name, role.Hash)
	}
	return nil
}
