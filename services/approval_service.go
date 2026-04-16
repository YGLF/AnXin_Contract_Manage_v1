package services

import (
	"contract-manage/models"
	"errors"
	"fmt"
	"time"
)

// 审批角色级别映射
// 数值越大，级别越高，可以审批比自己级别低的角色提交的申请
var ApprovalRoleLevels = map[string]int{
	"user":        1, // 普通用户
	"manager":     2, // 经理
	"admin":       3, // 管理员
	"audit_admin": 1, // 审计管理员（只能查看，不能审批）
}

// CanApproveRole 检查角色是否有权审批
// manager 可以审批 user 提交的合同
// admin 可以审批所有用户提交的合同
func CanApproveRole(approverRole, submitterRole string) bool {
	approverLevel := ApprovalRoleLevels[approverRole]
	submitterLevel := ApprovalRoleLevels[submitterRole]
	return approverLevel > submitterLevel
}

// ApprovalService 审批服务结构体，提供审批相关的业务逻辑
type ApprovalService struct {
	contractService *ContractService
}

// NewApprovalService 创建审批服务实例
// 初始化时创建关联的ContractService用于处理合同相关操作
func NewApprovalService() *ApprovalService {
	return &ApprovalService{
		contractService: NewContractService(),
	}
}

// GetPendingStatusChangesCount 获取待处理的状态变更请求数量
// 用于管理员查看待处理的合同状态变更申请数量
// 返回：待处理数量和错误信息
func (s *ApprovalService) GetPendingStatusChangesCount() (int, error) {
	requests, err := s.contractService.GetPendingStatusChangeRequests("admin")
	if err != nil {
		return 0, err
	}
	return len(requests), nil
}

// GetApprovalRecordByID 根据审批记录ID获取审批记录信息
// 参数：id - 审批记录ID
// 返回：包含审批人信息的审批记录对象和错误信息
func (s *ApprovalService) GetApprovalRecordByID(id uint) (*models.ApprovalRecord, error) {
	var record models.ApprovalRecord
	if err := models.DB.Preload("Approver").First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetApprovalRecords 获取指定合同的所有审批记录
// 参数：contractID - 合同ID
// 返回：按创建时间倒序排列的审批记录列表
func (s *ApprovalService) GetApprovalRecords(contractID uint) ([]models.ApprovalRecord, error) {
	var records []models.ApprovalRecord
	if err := models.DB.Where("contract_id = ?", contractID).Preload("Approver").Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// ApprovalRecordCreateInput 创建审批记录的输入结构体
type ApprovalRecordCreateInput struct {
	ContractID   uint   `json:"contract_id"`
	Level        int    `json:"level"`
	ApproverRole string `json:"approver_role"`
	Status       string `json:"status"`
	Comment      string `json:"comment"`
	TimeoutHours int    `json:"timeout_hours"` // 超时小时数，默认72小时
}

// CreateApprovalRecord 创建新的审批记录
// 功能说明：
//   - 创建审批记录并关联审批人
//   - 设置审批状态为待审批
//   - 自动设置审批截止时间
//
// 参数说明：
//   - input: 审批记录创建输入
//   - approverID: 审批人用户ID
//   - approverRole: 审批人角色
func (s *ApprovalService) CreateApprovalRecord(input ApprovalRecordCreateInput, approverID uint, approverRole string) (*models.ApprovalRecord, error) {
	record := models.ApprovalRecord{
		ContractID:   input.ContractID,
		ApproverID:   approverID,
		Level:        input.Level,
		ApproverRole: approverRole,
		Status:       models.ApprovalPending,
		Comment:      input.Comment,
	}

	// 设置审批截止时间
	timeoutHours := input.TimeoutHours
	if timeoutHours <= 0 {
		timeoutHours = models.DefaultApprovalTimeoutHours
	}
	dueAt := time.Now().Add(time.Duration(timeoutHours) * time.Hour)
	record.DueAt = &dueAt

	// 如果提供了状态，覆盖默认的待审批状态
	if input.Status != "" {
		record.Status = models.ApprovalStatus(input.Status)
	}

	if err := models.DB.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ApprovalRecordUpdateInput 更新审批记录的输入结构体
type ApprovalRecordUpdateInput struct {
	Status       string `json:"status" binding:"required"`
	Comment      string `json:"comment"`
	OperatorRole string `json:"operator_role"` // 操作人角色，用于权限验证
}

// UpdateApprovalRecord 更新审批记录（审批操作）
// 功能说明：
//   - 检查审批记录是否已处理，防止重复审批
//   - 验证审批人角色权限
//   - 更新审批状态和备注
//   - 如果提供了合同状态，同时更新合同状态
//   - 记录生命周期事件（审批通过/拒绝）
//
// 参数说明：
//   - id: 审批记录ID
//   - input: 包含审批状态和备注的输入
//   - contractStatus: 目标合同状态
//   - operatorID: 操作人ID
func (s *ApprovalService) UpdateApprovalRecord(id uint, input ApprovalRecordUpdateInput, contractStatus string, operatorID uint) (*models.ApprovalRecord, error) {
	record, err := s.GetApprovalRecordByID(id)
	if err != nil {
		return nil, err
	}

	// 检查审批记录是否已处理
	if record.Status != models.ApprovalPending {
		return nil, errors.New("this approval has already been processed")
	}

	// 获取合同信息，验证审批人角色权限
	var contract models.Contract
	if err := models.DB.First(&contract, record.ContractID).Error; err != nil {
		return nil, errors.New("contract not found")
	}

	// 验证审批人角色权限：只有比提交人级别高的角色才能审批
	if input.OperatorRole != "" && contract.CreatorID > 0 {
		var creator models.User
		if err := models.DB.First(&creator, contract.CreatorID).Error; err == nil {
			if !CanApproveRole(input.OperatorRole, string(creator.Role)) {
				return nil, errors.New("you don't have permission to approve this contract")
			}
		}
	}

	// 检查是否超时
	if record.IsApprovalExpired() {
		record.IsExpired = true
		record.Status = models.ApprovalRejected
		models.DB.Save(record)
		return nil, errors.New("approval has expired")
	}

	oldStatus := string(models.StatusPending)
	var newStatus string

	now := time.Now()
	record.Status = models.ApprovalStatus(input.Status)
	record.Comment = input.Comment
	record.ApprovedAt = &now

	if err := models.DB.Save(record).Error; err != nil {
		return nil, err
	}

	// 如果提供了合同状态，更新合同状态并记录生命周期事件
	if contractStatus != "" {
		oldStatus = string(contract.Status)
		contract.Status = models.ContractStatus(contractStatus)
		newStatus = contractStatus
		models.DB.Save(&contract)

		// 根据审批结果确定事件类型
		var eventType models.LifecycleEventType
		var description string
		if input.Status == "approved" {
			eventType = models.LifecycleApproved
			description = "审批通过"
		} else if input.Status == "rejected" {
			eventType = models.LifecycleRejected
			description = "审批拒绝"
		}

		// 记录生命周期事件
		if eventType != "" {
			s.contractService.AddLifecycleEvent(record.ContractID, LifecycleEventInput{
				EventType:   string(eventType),
				FromStatus:  oldStatus,
				ToStatus:    newStatus,
				Description: description,
			}, operatorID)
		}
	}

	return record, nil
}

// GetPendingApprovalsByRole 获取指定角色的待审批列表
// 功能说明：
//   - manager角色：查看草稿状态合同的待提交审批
//   - admin角色：查看待审批状态合同及其最新审批记录
//
// 参数说明：
//   - role: 审批人角色（manager/admin）
//   - userID: 当前用户ID
//
// 返回：待审批合同列表（包含合同和审批信息）
func (s *ApprovalService) GetPendingApprovalsByRole(role string, userID uint) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	var contracts []models.Contract
	query := models.DB.Preload("Customer").Order("created_at DESC")

	// 根据角色确定查询条件
	if role == "manager" {
		// 经理查看草稿状态的合同
		query = query.Where("status = ?", "draft")
		if err := query.Find(&contracts).Error; err != nil {
			return nil, err
		}
		for _, c := range contracts {
			results = append(results, map[string]interface{}{
				"id":               c.ID,
				"contract_no":      c.ContractNo,
				"title":            c.Title,
				"amount":           c.Amount,
				"status":           c.Status,
				"created_at":       c.CreatedAt,
				"customer":         c.Customer,
				"creator_id":       c.CreatorID,
				"contract_type_id": c.ContractTypeID,
				"approval_id":      uint(0),
			})
		}
	} else if role == "admin" {
		// 管理员查看待审批状态的合同
		query = query.Where("status = ?", "pending")
		if err := query.Find(&contracts).Error; err != nil {
			return nil, err
		}
		for _, c := range contracts {
			// 获取合同最新的审批记录
			var latestApproval models.ApprovalRecord
			models.DB.Where("contract_id = ?", c.ID).Order("created_at DESC").First(&latestApproval)
			results = append(results, map[string]interface{}{
				"id":               c.ID,
				"contract_no":      c.ContractNo,
				"title":            c.Title,
				"amount":           c.Amount,
				"status":           c.Status,
				"created_at":       c.CreatedAt,
				"customer":         c.Customer,
				"creator_id":       c.CreatorID,
				"contract_type_id": c.ContractTypeID,
				"approval_id":      latestApproval.ID,
			})
		}
	}

	return results, nil
}

// SubmitForApproval 提交合同进行审批
// 功能说明：
// - 将合同状态从草稿改为待审批
// - 记录生命周期事件（提交审批）
// - 通知相应审批人有新的待审批任务
//
// 参数：contractID - 合同ID, userID - 提交人ID
func (s *ApprovalService) SubmitForApproval(contractID uint, userID uint) error {
	var contract models.Contract
	if err := models.DB.First(&contract, contractID).Error; err != nil {
		return err
	}

	oldStatus := string(contract.Status)
	contract.Status = models.StatusPending
	if err := models.DB.Save(&contract).Error; err != nil {
		return err
	}

	// 记录生命周期事件
	s.contractService.AddLifecycleEvent(contractID, LifecycleEventInput{
		EventType:   string(models.LifecycleSubmitted),
		FromStatus:  oldStatus,
		ToStatus:    string(models.StatusPending),
		Description: "合同提交审批",
	}, userID)

	// 自动创建审批工作流
	workflowService := NewWorkflowService(models.DB)
	creatorRole := "sales"
	if contract.CreatorID > 0 {
		var creator models.User
		if err := models.DB.First(&creator, contract.CreatorID).Error; err == nil {
			creatorRole = string(creator.Role)
		}
	}
	_, err := workflowService.CreateWorkflow(uint64(contractID), uint64(userID), creatorRole)
	if err != nil {
		fmt.Printf("创建审批工作流失败: %v\n", err)
	}

	// 通知销售总监有待审批任务
	s.notifyApprover(contract, 1, "销售总监")

	return nil
}

func getApproverRole(level int) string {
	switch level {
	case 1:
		return "sales_director"
	case 2:
		return "tech_director"
	case 3:
		return "finance_director"
	default:
		return ""
	}
}

// notifyApprover 通知审批人有新的待审批任务
func (s *ApprovalService) notifyApprover(contract models.Contract, level int, roleName string) {
	// 查找该角色的用户
	var approver models.User
	if err := models.DB.Where("role = ?", getApproverRole(level)).First(&approver).Error; err != nil {
		return
	}

	notification := models.Notification{
		UserID:     approver.ID,
		ContractID: contract.ID,
		Type:       models.NotificationTypePendingApproval,
		Title:      "新的待审批任务",
		Content: fmt.Sprintf("您有一个待审批的合同：%s（%s），金额：¥%.2f，请尽快处理。",
			contract.Title, contract.ContractNo, contract.Amount),
		IsRead: false,
	}
	models.DB.Create(&notification)
}

// GetReminderByID 根据提醒ID获取提醒信息
// 参数：id - 提醒ID
// 返回：提醒对象和错误信息
func (s *ApprovalService) GetReminderByID(id uint) (*models.Reminder, error) {
	var reminder models.Reminder
	if err := models.DB.First(&reminder, id).Error; err != nil {
		return nil, err
	}
	return &reminder, nil
}

// GetReminders 获取合同的所有提醒记录
// 参数：contractID - 合同ID
// 返回：按提醒日期倒序排列的提醒列表
func (s *ApprovalService) GetReminders(contractID uint) ([]models.Reminder, error) {
	var reminders []models.Reminder
	if err := models.DB.Where("contract_id = ?", contractID).Order("reminder_date DESC").Find(&reminders).Error; err != nil {
		return nil, err
	}
	return reminders, nil
}

// ReminderCreateInput 创建提醒的输入结构体
type ReminderCreateInput struct {
	ContractID   uint      `json:"contract_id" binding:"required"`
	Type         string    `json:"type" binding:"required"`
	ReminderDate *JSONTime `json:"reminder_date" binding:"required"`
	DaysBefore   int       `json:"days_before" binding:"required"`
}

// CreateReminder 创建新的合同提醒
// 功能说明：
//   - 创建提醒记录
//   - 设置提醒日期
//   - 默认未发送状态
func (s *ApprovalService) CreateReminder(input ReminderCreateInput) (*models.Reminder, error) {
	reminder := models.Reminder{
		ContractID: input.ContractID,
		Type:       input.Type,
		DaysBefore: input.DaysBefore,
		IsSent:     false,
	}

	// 设置提醒日期
	if input.ReminderDate != nil && !input.ReminderDate.Time.IsZero() {
		reminder.ReminderDate = &input.ReminderDate.Time
	}

	if err := models.DB.Create(&reminder).Error; err != nil {
		return nil, err
	}
	return &reminder, nil
}

// UpdateReminderSent 标记提醒已发送
// 功能说明：
//   - 将提醒状态标记为已发送
//   - 记录发送时间
//
// 参数：id - 提醒ID
func (s *ApprovalService) UpdateReminderSent(id uint) error {
	reminder, err := s.GetReminderByID(id)
	if err != nil {
		return err
	}

	now := time.Now()
	reminder.IsSent = true
	reminder.SentAt = &now

	return models.DB.Save(reminder).Error
}

// GetExpiringContracts 获取即将到期的合同
// 功能说明：
//   - 查询指定天数内将到期的合同
//   - 只查询有效状态的合同
//
// 参数：days - 提前提醒天数
// 返回：即将到期的合同列表
func (s *ApprovalService) GetExpiringContracts(days int) ([]models.Contract, error) {
	today := time.Now()
	expiryDate := today.AddDate(0, 0, days)

	var contracts []models.Contract
	if err := models.DB.Where("end_date <= ? AND end_date >= ? AND status = ?",
		expiryDate, today, models.StatusActive).Find(&contracts).Error; err != nil {
		return nil, err
	}
	return contracts, nil
}

// Statistics 统计数据结构
type Statistics struct {
	TotalContracts      int64   `json:"total_contracts"`      // 合同总数
	ActiveContracts     int64   `json:"active_contracts"`     // 有效合同数
	PendingContracts    int64   `json:"pending_contracts"`    // 待审批合同数
	CompletedContracts  int64   `json:"completed_contracts"`  // 已完成合同数
	DraftContracts      int64   `json:"draft_contracts"`      // 草稿合同数
	TerminatedContracts int64   `json:"terminated_contracts"` // 已终止合同数
	TotalAmount         float64 `json:"total_amount"`         // 合同总金额
	ThisMonthContracts  int64   `json:"this_month_contracts"` // 本月新增合同数
	ThisMonthAmount     float64 `json:"this_month_amount"`    // 本月合同金额
	ExpiringSoon        int     `json:"expiring_soon"`        // 即将到期合同数
}

// GetStatistics 获取合同统计数据
// 功能说明：
//   - 统计各状态的合同数量
//   - 计算合同总金额
//   - 统计本月新增合同和金额
//   - 统计30天内即将到期的合同数量
//
// 返回：统计数据对象
func (s *ApprovalService) GetStatistics() (*Statistics, error) {
	today := time.Now()
	// 计算本月起始日期
	thisMonthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local)

	stats := &Statistics{}

	// 统计各状态合同数量
	models.DB.Model(&models.Contract{}).Count(&stats.TotalContracts)
	models.DB.Model(&models.Contract{}).Where("status = ?", models.StatusActive).Count(&stats.ActiveContracts)
	models.DB.Model(&models.Contract{}).Where("status = ?", models.StatusPending).Count(&stats.PendingContracts)
	models.DB.Model(&models.Contract{}).Where("status = ?", models.StatusCompleted).Count(&stats.CompletedContracts)
	models.DB.Model(&models.Contract{}).Where("status = ?", models.StatusDraft).Count(&stats.DraftContracts)
	models.DB.Model(&models.Contract{}).Where("status = ?", models.StatusTerminated).Count(&stats.TerminatedContracts)

	// 计算合同总金额
	var totalAmount *float64
	models.DB.Model(&models.Contract{}).Where("amount IS NOT NULL").Select("SUM(amount)").Scan(&totalAmount)
	if totalAmount != nil {
		stats.TotalAmount = *totalAmount
	}

	// 统计本月新增合同数
	models.DB.Model(&models.Contract{}).Where("created_at >= ?", thisMonthStart).Count(&stats.ThisMonthContracts)

	// 计算本月合同金额
	var thisMonthAmount *float64
	models.DB.Model(&models.Contract{}).Where("created_at >= ? AND amount IS NOT NULL", thisMonthStart).Select("SUM(amount)").Scan(&thisMonthAmount)
	if thisMonthAmount != nil {
		stats.ThisMonthAmount = *thisMonthAmount
	}

	// 统计30天内即将到期的合同
	expiring, _ := s.GetExpiringContracts(30)
	stats.ExpiringSoon = len(expiring)

	return stats, nil
}

// ProcessExpiredApprovals 自动处理超时的审批
// 功能说明：
//   - 查找所有超时的待审批记录
//   - 将其状态更新为已拒绝
//   - 更新关联合同状态为草稿（可重新提交）
//   - 记录生命周期事件
//
// 返回：处理的记录数和错误信息
func (s *ApprovalService) ProcessExpiredApprovals() (int, error) {
	now := time.Now()

	// 查找超时的待审批记录
	var expiredRecords []models.ApprovalRecord
	if err := models.DB.Where("status = ? AND due_at < ?", models.ApprovalPending, now).Find(&expiredRecords).Error; err != nil {
		return 0, err
	}

	processedCount := 0
	for _, record := range expiredRecords {
		// 更新审批记录为超时拒绝
		record.Status = models.ApprovalRejected
		record.IsExpired = true
		record.ApprovedAt = &now
		record.Comment = "审批超时自动拒绝"
		models.DB.Save(&record)

		// 将合同状态改回草稿
		var contract models.Contract
		if err := models.DB.First(&contract, record.ContractID).Error; err == nil {
			oldStatus := string(contract.Status)
			contract.Status = models.StatusDraft
			models.DB.Save(&contract)

			// 记录生命周期事件
			s.contractService.AddLifecycleEvent(record.ContractID, LifecycleEventInput{
				EventType:   string(models.LifecycleRejected),
				FromStatus:  oldStatus,
				ToStatus:    string(models.StatusDraft),
				Description: fmt.Sprintf("审批超时（超过%d小时），合同退回草稿", models.DefaultApprovalTimeoutHours),
			}, 0) // 系统自动操作
		}

		processedCount++
	}

	return processedCount, nil
}

// RollbackApproval 回退审批到上一步
// 功能说明：
//   - 将当前审批记录状态改回待审批
//   - 重置审批时间和意见
//   - 将合同状态改回待审批
//
// 参数说明：
//   - id: 审批记录ID
//   - operatorID: 操作人ID
//
// 仅管理员可以执行回退操作
func (s *ApprovalService) RollbackApproval(id uint, operatorID uint, operatorRole string) (*models.ApprovalRecord, error) {
	// 仅管理员可以回退审批
	if operatorRole != "admin" {
		return nil, errors.New("only admin can rollback approvals")
	}

	record, err := s.GetApprovalRecordByID(id)
	if err != nil {
		return nil, err
	}

	// 只能回退已处理的审批记录
	if record.Status == models.ApprovalPending {
		return nil, errors.New("approval is still pending, no need to rollback")
	}

	// 回退审批记录
	record.Status = models.ApprovalPending
	record.IsExpired = false
	record.ApprovedAt = nil
	record.Comment = ""

	// 重置截止时间
	dueAt := time.Now().Add(time.Duration(models.DefaultApprovalTimeoutHours) * time.Hour)
	record.DueAt = &dueAt

	if err := models.DB.Save(record).Error; err != nil {
		return nil, err
	}

	// 将合同状态改回待审批
	var contract models.Contract
	if err := models.DB.First(&contract, record.ContractID).Error; err == nil {
		oldStatus := string(contract.Status)
		contract.Status = models.StatusPending
		models.DB.Save(&contract)

		// 记录生命周期事件
		s.contractService.AddLifecycleEvent(record.ContractID, LifecycleEventInput{
			EventType:   string(models.LifecycleSubmitted),
			FromStatus:  oldStatus,
			ToStatus:    string(models.StatusPending),
			Description: "审批回退，合同重新进入待审批状态",
		}, operatorID)
	}

	return record, nil
}

// GetApprovalStatusInfo 获取审批状态详细信息
// 用于前端展示审批状态和剩余时间
func (s *ApprovalService) GetApprovalStatusInfo(id uint) (map[string]interface{}, error) {
	record, err := s.GetApprovalRecordByID(id)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"id":          record.ID,
		"status":      record.Status,
		"status_text": getApprovalStatusText(record.Status),
		"created_at":  record.CreatedAt,
		"due_at":      record.DueAt,
		"is_expired":  record.IsApprovalExpired(),
		"approved_at": record.ApprovedAt,
	}

	// 计算剩余时间
	if record.DueAt != nil && record.Status == models.ApprovalPending {
		remaining := time.Until(*record.DueAt)
		if remaining > 0 {
			hours := int(remaining.Hours())
			minutes := int(remaining.Minutes()) % 60
			info["remaining_time"] = fmt.Sprintf("%d小时%d分钟", hours, minutes)
			info["remaining_hours"] = remaining.Hours()
		} else {
			info["remaining_time"] = "已超时"
			info["remaining_hours"] = 0
		}
	}

	return info, nil
}

// getApprovalStatusText 获取审批状态中文文本
func getApprovalStatusText(status models.ApprovalStatus) string {
	switch status {
	case models.ApprovalPending:
		return "待审批"
	case models.ApprovalApproved:
		return "已通过"
	case models.ApprovalRejected:
		return "已拒绝"
	default:
		return "未知状态"
	}
}
