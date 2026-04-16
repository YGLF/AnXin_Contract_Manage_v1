package services

import (
	"contract-manage/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type rawWfNode struct {
	ID           uint64
	WfID         uint64
	ContractID   uint64
	ApproverID   *uint64
	ApproverRole string
	Level        int
	Status       string
	Comment      *string
	Hash         *string
	ApprovedAt   *time.Time
	CreatedAt    time.Time
}

// 审批流程配置
// 定义标准的三级审批流程
var ApprovalLevels = []struct {
	Level       int
	Role        models.UserRole
	Name        string
	Description string
}{
	{1, models.RoleSalesDirector, "销售总监审批", "销售总监审核合同内容和报价"},
	{2, models.RoleTechDirector, "技术总监审批", "技术总监审核技术方案和可行性"},
	{3, models.RoleFinanceDirector, "财务总监审批", "财务总监审核财务预算和风险"},
}

// 审批级别常量
const (
	ApprovalLevelSales   = iota + 1 // 1: 销售总监审批
	ApprovalLevelTech               // 2: 技术总监审批
	ApprovalLevelFinance            // 3: 财务总监审批
)

// GetNextApprovalRole 获取下一审批级别和角色
// 参数：currentLevel - 当前审批级别
// 返回：下一审批级别(如果是最后一级则返回0)和角色
func GetNextApprovalRole(currentLevel int) (int, models.UserRole) {
	switch currentLevel {
	case ApprovalLevelSales:
		return ApprovalLevelTech, models.RoleTechDirector
	case ApprovalLevelTech:
		return ApprovalLevelFinance, models.RoleFinanceDirector
	case ApprovalLevelFinance:
		return 0, "" // 审批流程结束
	default:
		return 0, ""
	}
}

// GetCurrentApprovalRole 获取指定审批级别对应的角色
func GetCurrentApprovalRole(level int) models.UserRole {
	switch level {
	case ApprovalLevelSales:
		return models.RoleSalesDirector
	case ApprovalLevelTech:
		return models.RoleTechDirector
	case ApprovalLevelFinance:
		return models.RoleFinanceDirector
	default:
		return ""
	}
}

// IsLastApprovalLevel 判断是否为最后一级审批
func IsLastApprovalLevel(level int) bool {
	return level >= ApprovalLevelFinance
}

// WorkflowService 工作流服务结构体，提供审批工作流相关的业务逻辑
type WorkflowService struct {
	db *gorm.DB
}

// NewWorkflowService 创建工作流服务实例
// 参数：db - GORM数据库实例
func NewWorkflowService(db *gorm.DB) *WorkflowService {
	return &WorkflowService{db: db}
}

// CreateWorkflow 为合同创建审批工作流
// 功能说明：
//   - 如果合同已有被拒绝的工作流，则重新激活该工作流
//   - 否则创建新的工作流记录
//   - 设置当前审批级别为1（销售总监审批）
//   - 创建三级审批流程：销售总监 -> 技术总监 -> 财务总监
//
// 参数说明：
//   - contractID: 关联的合同ID
//   - creatorID: 创建人ID（销售人员）
//   - creatorRole: 创建人角色
//
// 返回：创建的工作流对象和错误信息
func (s *WorkflowService) CreateWorkflow(contractID uint64, creatorID uint64, creatorRole string) (*models.ApprovalWorkflow, error) {
	// 检查是否已有该合同的工作流（可能是被拒绝后重置的）
	var existingWorkflow models.ApprovalWorkflow
	if err := s.db.Where("contract_id = ?", contractID).First(&existingWorkflow).Error; err == nil {
		// 找到已存在的工作流，重新激活它
		s.db.Model(&existingWorkflow).Updates(map[string]interface{}{
			"current_level": ApprovalLevelSales,
			"status":        models.WorkflowStatusPending,
		})
		// 重置所有审批节点为pending
		s.db.Table("workflow_approvals").Where("workflow_id = ?", existingWorkflow.ID).Updates(map[string]interface{}{
			"status":      models.WorkflowStatusPending,
			"approver_id": nil,
			"comment":     "",
			"approved_at": nil,
		})
		// 更新合同状态为待审批
		s.db.Model(&models.Contract{}).Where("id = ?", contractID).Update("status", models.StatusPending)
		return &existingWorkflow, nil
	}

	// 没有现有工作流，创建新的
	workflow := &models.ApprovalWorkflow{
		ContractID:   contractID,
		CreatorID:    creatorID,
		CurrentLevel: ApprovalLevelSales,
		MaxLevel:     ApprovalLevelFinance, // 3级审批
		Status:       models.WorkflowStatusPending,
		CreatorRole:  creatorRole,
	}

	// 创建工作流记录
	if err := s.db.Create(workflow).Error; err != nil {
		return nil, err
	}

	// 创建三级审批人配置
	approvers := []models.WfNode{
		{
			WfID:         workflow.ID,
			ContractID:   contractID,
			ApproverRole: string(models.RoleSalesDirector),
			Level:        ApprovalLevelSales,
			Status:       models.WorkflowStatusPending,
		},
		{
			WfID:         workflow.ID,
			ContractID:   contractID,
			ApproverRole: string(models.RoleTechDirector),
			Level:        ApprovalLevelTech,
			Status:       models.WorkflowStatusPending,
		},
		{
			WfID:         workflow.ID,
			ContractID:   contractID,
			ApproverRole: string(models.RoleFinanceDirector),
			Level:        ApprovalLevelFinance,
			Status:       models.WorkflowStatusPending,
		},
	}

	// 批量创建审批人记录
	if err := s.db.Create(&approvers).Error; err != nil {
		return nil, err
	}

	// 更新合同状态为待审批
	s.db.Model(&models.Contract{}).Where("id = ?", contractID).Update("status", models.StatusPending)

	return workflow, nil
}

// GetWorkflowByContractID 根据合同ID获取工作流信息
// 功能说明：
// - 查询合同关联的工作流
// - 预加载所有审批人的详细信息
//
// 参数：contractID - 合同ID
// 返回：工作流对象（包含审批人列表）和错误信息
func (s *WorkflowService) GetWorkflowByContractID(contractID uint64) (*models.ApprovalWorkflow, error) {
	var workflow models.ApprovalWorkflow
	if err := s.db.Where("contract_id = ?", contractID).First(&workflow).Error; err != nil {
		return nil, err
	}

	// 手动加载审批节点列表
	var approvals []models.WfNode
	if err := s.db.Where("workflow_id = ?", workflow.ID).Find(&approvals).Error; err == nil {
		workflow.ID = workflow.ID // 确保有ID
	}

	return &workflow, nil
}

// GetWorkflowByWorkflowID 根据工作流ID获取工作流信息
func (s *WorkflowService) GetWorkflowByWorkflowID(workflowID uint64, workflow *models.ApprovalWorkflow) error {
	return s.db.First(workflow, workflowID).Error
}

// GetApprovalsByWorkflowID 获取工作流的所有审批节点
func (s *WorkflowService) GetApprovalsByWorkflowID(workflowID uint64) ([]models.WfNode, error) {
	var approvals []models.WfNode
	err := s.db.Where("workflow_id = ?", workflowID).Order("level ASC").Find(&approvals).Error
	return approvals, err
}

// Approve 审批通过操作
// 功能说明：
// - 更新指定层级的审批记录为已通过状态
// - 记录审批人和审批时间
// - 如果是最后一层（财务总监），更新工作流状态为完成，合同生效
// - 否则，升级到下一审批层级
//
// 参数说明：
//   - workflowID: 工作流ID
//   - contractID: 合同ID
//   - level: 当前审批层级
//   - approverID: 审批人ID
//   - approverRole: 审批人角色
//   - comment: 审批备注
func (s *WorkflowService) Approve(workflowID uint64, contractID uint64, level int, approverID uint64, approverRole string, comment string) error {
	// 验证审批人角色是否与当前审批级别匹配
	currentRequiredRole := GetCurrentApprovalRole(level)
	if models.UserRole(approverRole) != currentRequiredRole {
		return fmt.Errorf("审批人角色不匹配，当前需要%s审批", currentRequiredRole)
	}

	// 查询指定层级的审批记录（使用rawWfNode避免GORM关系解析问题）
	var approval rawWfNode
	if err := s.db.Table("workflow_approvals").Where("workflow_id = ? AND level = ?", workflowID, level).First(&approval).Error; err != nil {
		return err
	}

	// 检查审批记录是否已被处理
	if approval.Status != models.WorkflowStatusPending {
		return fmt.Errorf("该审批级别已处理")
	}

	now := time.Now()
	approverIDUint := uint64(approverID)
	// 更新审批记录为通过状态
	if err := s.db.Table("workflow_approvals").Where("id = ?", approval.ID).Updates(map[string]interface{}{
		"status":      models.WorkflowStatusApproved,
		"approver_id": approverIDUint,
		"comment":     comment,
		"approved_at": now,
	}).Error; err != nil {
		return err
	}

	// 获取工作流信息
	var workflow models.ApprovalWorkflow
	if err := s.db.First(&workflow, workflowID).Error; err != nil {
		return err
	}

	// 判断是否到达最后一层（财务总监）
	if level >= ApprovalLevelFinance {
		// 最后一层审批通过，工作流完成
		s.db.Model(&workflow).Updates(map[string]interface{}{
			"status": models.WorkflowStatusCompleted,
		})

		// 获取合同信息，检查结束时间
		var contract models.Contract
		if err := s.db.First(&contract, contractID).Error; err == nil {
			// 如果合同结束日期已过期，直接归档
			// 否则设置为执行中状态
			if contract.EndDate != nil && contract.EndDate.Before(time.Now()) {
				s.db.Model(&contract).Updates(map[string]interface{}{
					"status": models.StatusArchived,
				})
				s.addLifecycleEvent(contractID, "workflow_completed", string(models.StatusActive), string(models.StatusArchived), "审批流程完成，合同已自动归档", approverID)
			} else if contract.EndDate != nil {
				// 合同未过期，设置为执行中状态
				s.db.Model(&contract).Updates(map[string]interface{}{
					"status": models.StatusInProgress,
				})
				// 记录生命周期事件
				s.addLifecycleEvent(contractID, "workflow_completed", "", string(models.StatusInProgress), "审批流程完成，合同开始执行", approverID)
			} else {
				// 无结束日期，设置为生效状态
				s.db.Model(&contract).Updates(map[string]interface{}{
					"status": models.StatusActive,
				})
				s.addLifecycleEvent(contractID, "workflow_completed", "", string(models.StatusActive), "审批流程完成，合同已生效", approverID)
			}
		} else {
			// 更新合同状态为已生效
			s.db.Model(&models.Contract{}).Where("id = ?", contractID).Updates(map[string]interface{}{
				"status": models.StatusActive,
			})
			s.addLifecycleEvent(contractID, "workflow_completed", "", string(models.StatusActive), "审批流程完成，合同已生效", approverID)
		}

		// 通知销售人员审批全部通过
		s.notifySalesApproved(contractID, workflowID)
	} else {
		// 未到最后一层，升级到下一审批层级
		nextLevel, nextRole := GetNextApprovalRole(level)
		s.db.Model(&workflow).Updates(map[string]interface{}{
			"current_level": nextLevel,
			"status":        models.WorkflowStatusPending,
		})

		// 通知下一审批人
		s.notifyNextApprover(contractID, workflowID, nextRole)

		// 记录生命周期事件
		s.addLifecycleEvent(contractID, "workflow_approved", "", "", fmt.Sprintf("第%d级审批通过，流转到%s审批", level, nextRole), approverID)
	}

	return nil
}

// Reject 审批拒绝操作
// 功能说明：
//   - 更新指定层级的审批记录为已拒绝状态
//   - 将工作流重置到销售总监级别（重新开始审批流程）
//   - 将合同状态改回待提交状态
//   - 通知销售人员需要重新处理
//
// 参数说明：
//   - workflowID: 工作流ID
//   - level: 当前审批层级
//   - approverID: 审批人ID
//   - approverRole: 审批人角色
//   - comment: 拒绝原因
func (s *WorkflowService) Reject(workflowID uint64, level int, approverID uint64, approverRole string, comment string) error {
	// 验证审批人角色是否与当前审批级别匹配
	currentRequiredRole := GetCurrentApprovalRole(level)
	if models.UserRole(approverRole) != currentRequiredRole {
		return fmt.Errorf("审批人角色不匹配，当前需要%s审批", currentRequiredRole)
	}

	// 查询指定层级的审批记录（使用rawWfNode避免GORM关系解析问题）
	var approval rawWfNode
	if err := s.db.Table("workflow_approvals").Where("workflow_id = ? AND level = ?", workflowID, level).First(&approval).Error; err != nil {
		return err
	}

	now := time.Now()
	approverIDUint := uint64(approverID)
	// 更新当前审批记录为拒绝状态
	if err := s.db.Table("workflow_approvals").Where("id = ?", approval.ID).Updates(map[string]interface{}{
		"status":      models.WorkflowStatusRejected,
		"approver_id": approverIDUint,
		"comment":     comment,
		"approved_at": now,
	}).Error; err != nil {
		return err
	}

	// 获取工作流信息
	var workflow models.ApprovalWorkflow
	if err := s.db.First(&workflow, workflowID).Error; err != nil {
		return err
	}

	// 重置工作流：将current_level重置为1（从销售总监重新开始）
	s.db.Model(&workflow).Updates(map[string]interface{}{
		"current_level": 1,
		"status":        models.WorkflowStatusPending,
	})

	// 重置所有审批节点为pending状态
	s.db.Table("workflow_approvals").Where("workflow_id = ?", workflowID).Updates(map[string]interface{}{
		"status":      models.WorkflowStatusPending,
		"approver_id": nil,
		"comment":     "",
		"approved_at": nil,
	})

	// 将合同状态改为草稿（等待销售重新修改提交）
	s.db.Model(&models.Contract{}).Where("id = ?", workflow.ContractID).Update("status", models.StatusDraft)

	// 记录生命周期事件
	s.addLifecycleEvent(workflow.ContractID, "workflow_rejected", string(models.StatusPending), string(models.StatusDraft), fmt.Sprintf("审批被拒绝（%s）：%s，请重新处理", currentRequiredRole, comment), approverID)

	// 通知销售人员审批被拒绝
	s.notifySalesRejected(workflow.ContractID, workflowID, approverRole, comment)

	return nil
}

// notifySalesRejected 通知销售人员审批被拒绝
func (s *WorkflowService) notifySalesRejected(contractID uint64, workflowID uint64, rejectedByRole string, reason string) {
	// 获取合同信息
	var contract models.Contract
	if err := s.db.First(&contract, contractID).Error; err != nil {
		return
	}

	// 获取合同创建者（销售人员）
	var creator models.User
	if err := s.db.First(&creator, contract.CreatorID).Error; err != nil {
		return
	}

	// 获取拒绝者的角色名称
	rejectedByName := ""
	switch rejectedByRole {
	case string(models.RoleTechDirector):
		rejectedByName = "技术总监"
	case string(models.RoleFinanceDirector):
		rejectedByName = "财务总监"
	default:
		rejectedByName = "审批人"
	}

	// 创建通知
	notification := models.Notification{
		UserID:     creator.ID,
		ContractID: uint(contractID),
		WorkflowID: uint(workflowID),
		Role:       string(models.RoleSales),
		Type:       models.NotificationTypeRejected,
		Title:      "合同审批被拒绝",
		Content: fmt.Sprintf("您的合同「%s」（%s）被%s拒绝。\n拒绝原因：%s\n请修改后重新提交审批。",
			contract.Title, contract.ContractNo, rejectedByName, reason),
		IsRead: false,
	}
	s.db.Create(&notification)
}

// GetPendingApprovals 获取指定角色的待审批列表
// 功能说明：
// - 查询指定角色需要审批的记录
// - 只返回当前审批级别且工作流处于待审批状态的记录
// - 预加载审批人、工作流和合同信息
//
// 参数：role - 审批人角色
// 返回：该角色需要处理的待审批列表
func (s *WorkflowService) GetPendingApprovals(role string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)
	var sqlQuery string
	var args []interface{}

	userRole := models.UserRole(role)

	// 使用原始SQL查询避免GORM关系问题
	// 超级管理员/合同管理员看到所有待审批合同，每合同一条记录
	if role == "admin" || role == "contract_admin" {
		sqlQuery = `SELECT w.id as workflow_id, w.contract_id, w.current_level, w.max_level, w.status as workflow_status, w.creator_role,
			c.contract_no, c.title, c.amount, c.status as contract_status, c.created_at
			FROM approval_workflows w
			JOIN contracts c ON c.id = w.contract_id
			WHERE w.status = ? AND c.status != 'draft' AND c.status != 'active'
			ORDER BY w.created_at DESC`
		args = []interface{}{models.WorkflowStatusPending}

		sqlRows, err := s.db.Raw(sqlQuery, args...).Rows()
		if err != nil {
			return nil, err
		}
		defer sqlRows.Close()

		for sqlRows.Next() {
			var wfID, contractID, currentLevel, maxLevel int
			var workflowStatus, contractStatus, contractNo, title, creatorRole string
			var amount float64
			var createdAt time.Time
			if err := sqlRows.Scan(&wfID, &contractID, &currentLevel, &maxLevel, &workflowStatus, &creatorRole, &contractNo, &title, &amount, &contractStatus, &createdAt); err != nil {
				continue
			}

			var contract models.Contract
			s.db.Preload("Customer").First(&contract, contractID)

			var creator *models.User
			if contract.CreatorID > 0 {
				var user models.User
				if err := s.db.First(&user, contract.CreatorID).Error; err == nil {
					creator = &user
				}
			}

			results = append(results, map[string]interface{}{
				"workflow_id":     wfID,
				"contract_id":     contractID,
				"contract_no":     contractNo,
				"title":           title,
				"amount":          amount,
				"status":          contractStatus,
				"level":           currentLevel,
				"level_name":      GetLevelName(currentLevel),
				"max_level":       maxLevel,
				"workflow_status": workflowStatus,
				"customer":        contract.Customer,
				"creator":         creator,
				"created_at":      createdAt,
			})
		}
		return results, nil
	}

	// 普通审批角色，按角色过滤
	var approverRole string
	if userRole == models.RoleSalesDirector {
		approverRole = "sales_director"
	} else if userRole == models.RoleTechDirector {
		approverRole = "tech_director"
	} else if userRole == models.RoleFinanceDirector {
		approverRole = "finance_director"
	} else {
		return results, nil
	}

	sqlQuery = `SELECT wa.id, wa.workflow_id, wa.contract_id, wa.approver_id, wa.approver_role, wa.level, wa.status, wa.comment, wa.hash, wa.approved_at, wa.created_at 
		FROM workflow_approvals wa
		JOIN approval_workflows w ON w.id = wa.workflow_id
		JOIN contracts c ON c.id = wa.contract_id
		WHERE wa.status = ? AND wa.approver_role = ? AND w.status = ? AND w.current_level = wa.level AND c.status != 'draft'`
	args = []interface{}{models.WorkflowStatusPending, approverRole, models.WorkflowStatusPending}

	sqlRows, err := s.db.Raw(sqlQuery, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	for sqlRows.Next() {
		var node rawWfNode
		if err := sqlRows.Scan(&node.ID, &node.WfID, &node.ContractID, &node.ApproverID, &node.ApproverRole, &node.Level, &node.Status, &node.Comment, &node.Hash, &node.ApprovedAt, &node.CreatedAt); err != nil {
			continue
		}

		var contract models.Contract
		if err := s.db.Preload("Customer").First(&contract, node.ContractID).Error; err != nil {
			continue
		}

		var approver *models.User
		if node.ApproverID != nil {
			var user models.User
			if err := s.db.First(&user, *node.ApproverID).Error; err == nil {
				approver = &user
			}
		}

		var creator *models.User
		if contract.CreatorID > 0 {
			var user models.User
			if err := s.db.First(&user, contract.CreatorID).Error; err == nil {
				creator = &user
			}
		}

		results = append(results, map[string]interface{}{
			"approval_id": node.ID,
			"workflow_id": node.WfID,
			"contract_id": contract.ID,
			"contract_no": contract.ContractNo,
			"title":       contract.Title,
			"amount":      contract.Amount,
			"status":      contract.Status,
			"level":       node.Level,
			"level_name":  GetLevelName(node.Level),
			"customer":    contract.Customer,
			"approver":    approver,
			"creator":     creator,
			"created_at":  contract.CreatedAt,
		})
	}

	return results, nil
}

// GetLevelName 获取审批级别名称
func GetLevelName(level int) string {
	switch level {
	case ApprovalLevelSales:
		return "销售总监审批"
	case ApprovalLevelTech:
		return "技术总监审批"
	case ApprovalLevelFinance:
		return "财务总监审批"
	default:
		return "未知级别"
	}
}

// addLifecycleEvent 添加生命周期事件
func (s *WorkflowService) addLifecycleEvent(contractID uint64, eventType, fromStatus, toStatus, description string, operatorID uint64) {
	event := models.ContractLifecycleEvent{
		ContractID:  uint(contractID),
		EventType:   models.LifecycleEventType(eventType),
		FromStatus:  fromStatus,
		ToStatus:    toStatus,
		Description: description,
		OperatorID:  uint(operatorID),
	}
	s.db.Create(&event)
}

// GetWorkflowStatus 获取合同的工作流状态详情
func (s *WorkflowService) GetWorkflowStatus(contractID uint64) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"has_workflow":  false,
		"status":        "",
		"current_level": 0,
		"max_level":     3,
		"nodes":         []map[string]interface{}{},
	}

	var workflow models.ApprovalWorkflow
	if err := s.db.Where("contract_id = ?", contractID).First(&workflow).Error; err != nil {
		return result, nil
	}

	result["has_workflow"] = true
	result["status"] = workflow.Status
	result["current_level"] = workflow.CurrentLevel
	result["max_level"] = workflow.MaxLevel

	// 获取所有审批节点（使用rawWfNode避免GORM关系解析问题）
	var nodes []rawWfNode
	s.db.Table("workflow_approvals").Where("workflow_id = ?", workflow.ID).Order("level ASC").Find(&nodes)

	nodeList := make([]map[string]interface{}, 0, len(nodes))
	pendingApprovers := []string{}

	for _, node := range nodes {
		nodeInfo := map[string]interface{}{
			"level":         node.Level,
			"level_name":    GetLevelName(node.Level),
			"approver_role": node.ApproverRole,
			"status":        node.Status,
			"comment":       node.Comment,
			"approved_at":   node.ApprovedAt,
		}

		// 获取审批人信息
		if node.ApproverID != nil {
			var user models.User
			if err := s.db.First(&user, *node.ApproverID).Error; err == nil {
				nodeInfo["approver_name"] = user.FullName
				nodeInfo["approver_id"] = user.ID
			}
		}

		// 记录待审批的角色
		if node.Status == "pending" {
			pendingApprovers = append(pendingApprovers, models.GetRoleDisplayName(models.UserRole(node.ApproverRole)))
		}

		nodeList = append(nodeList, nodeInfo)
	}

	result["nodes"] = nodeList
	result["pending_approvers"] = pendingApprovers

	return result, nil
}

// SendApprovalReminder 发送审批提醒
func (s *WorkflowService) SendApprovalReminder(contractID uint64, operatorID uint64) error {
	var workflow models.ApprovalWorkflow
	if err := s.db.Where("contract_id = ?", contractID).First(&workflow).Error; err != nil {
		return fmt.Errorf("工作流不存在")
	}

	if workflow.Status == "completed" || workflow.Status == "rejected" {
		return fmt.Errorf("合同审批已完成或已拒绝，无法催办")
	}

	// 获取合同信息
	var contract models.Contract
	if err := s.db.First(&contract, contractID).Error; err != nil {
		return fmt.Errorf("合同不存在")
	}

	// 获取当前待审批的节点（使用rawWfNode避免GORM关系解析问题）
	var pendingNodes []rawWfNode
	s.db.Table("workflow_approvals").Where("workflow_id = ? AND status = ?", workflow.ID, "pending").Find(&pendingNodes)

	if len(pendingNodes) == 0 {
		// 检查工作流是否已完成
		if workflow.CurrentLevel > workflow.MaxLevel {
			return fmt.Errorf("审批流程已完成")
		}
		return fmt.Errorf("当前没有待审批的节点，请等待上一级审批完成")
	}

	// 为每个待审批角色发送通知
	for _, node := range pendingNodes {
		// 查找该角色的用户
		var users []models.User
		s.db.Where("role = ? AND is_active = ?", node.ApproverRole, true).Find(&users)

		for _, user := range users {
			notification := models.Notification{
				UserID:     user.ID,
				ContractID: uint(contractID),
				WorkflowID: uint(workflow.ID),
				Role:       node.ApproverRole,
				Type:       models.NotificationTypeApprovalReminder,
				Title:      "合同审批提醒",
				Content: fmt.Sprintf("您有一个合同等待审批：%s（%s），金额：%.2f元。请尽快处理。",
					contract.Title, contract.ContractNo, contract.Amount),
				IsRead: false,
			}
			s.db.Create(&notification)
		}
	}

	return nil
}

// notifyNextApprover 通知下一审批人有新的审批任务
func (s *WorkflowService) notifyNextApprover(contractID uint64, workflowID uint64, nextRole models.UserRole) {
	// 获取合同信息
	var contract models.Contract
	if err := s.db.First(&contract, contractID).Error; err != nil {
		return
	}

	// 查找该角色的所有用户
	var users []models.User
	s.db.Where("role = ? AND is_active = ?", nextRole, true).Find(&users)

	for _, user := range users {
		notification := models.Notification{
			UserID:     user.ID,
			ContractID: uint(contractID),
			WorkflowID: uint(workflowID),
			Role:       string(nextRole),
			Type:       models.NotificationTypeApprovalReminder,
			Title:      "合同待审批",
			Content: fmt.Sprintf("您有一个合同等待审批：\n合同名称：%s\n合同编号：%s\n合同金额：¥%.2f\n请及时处理。",
				contract.Title, contract.ContractNo, contract.Amount),
			IsRead: false,
		}
		s.db.Create(&notification)
	}
}

// notifySalesApproved 通知销售人员审批全部通过
func (s *WorkflowService) notifySalesApproved(contractID uint64, workflowID uint64) {
	// 获取合同信息
	var contract models.Contract
	if err := s.db.First(&contract, contractID).Error; err != nil {
		return
	}

	// 获取合同创建者
	var creator models.User
	if err := s.db.First(&creator, contract.CreatorID).Error; err != nil {
		return
	}

	notification := models.Notification{
		UserID:     creator.ID,
		ContractID: uint(contractID),
		WorkflowID: uint(workflowID),
		Role:       string(models.RoleSales),
		Type:       models.NotificationTypeApproved,
		Title:      "合同审批全部通过",
		Content: fmt.Sprintf("您创建的合同「%s」（%s）已通过全部审批流程。\n合同金额：¥%.2f\n合同已归档生效。",
			contract.Title, contract.ContractNo, contract.Amount),
		IsRead: false,
	}
	s.db.Create(&notification)
}

// GetUserNotifications 获取用户的所有通知
func (s *WorkflowService) GetUserNotifications(userID uint64) ([]models.Notification, error) {
	var notifications []models.Notification
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

// MarkNotificationRead 标记通知为已读
func (s *WorkflowService) MarkNotificationRead(notificationID uint64, userID uint64) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

// GetUnreadNotificationCount 获取未读通知数量
func (s *WorkflowService) GetUnreadNotificationCount(userID uint64) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// DeleteNotification 删除指定通知
func (s *WorkflowService) DeleteNotification(notificationID uint64, userID uint64) error {
	return s.db.Where("id = ? AND user_id = ?", notificationID, userID).Delete(&models.Notification{}).Error
}

// DeleteAllNotifications 删除当前用户所有通知
func (s *WorkflowService) DeleteAllNotifications(userID uint64) error {
	return s.db.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}

// GetApprovalStats 获取审批统计数据
// 返回各审批级别的待审批数量和被拒绝数量
func (s *WorkflowService) GetApprovalStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"pendingCount":   0,
		"salesPending":   0,
		"techPending":    0,
		"financePending": 0,
		"rejectedCount":  0,
		"approvedCount":  0,
		"activeCount":    0,
		"completedCount": 0,
	}

	// 统计各审批级别待审批数量
	var count int64
	s.db.Model(&models.WfNode{}).
		Joins("JOIN approval_workflows ON approval_workflows.id = workflow_approvals.workflow_id").
		Where("workflow_approvals.status = ? AND approval_workflows.status = ? AND workflow_approvals.level = 1 AND approval_workflows.current_level = 1",
			models.WorkflowStatusPending, models.WorkflowStatusPending).
		Count(&count)
	stats["salesPending"] = count

	s.db.Model(&models.WfNode{}).
		Joins("JOIN approval_workflows ON approval_workflows.id = workflow_approvals.workflow_id").
		Where("workflow_approvals.status = ? AND approval_workflows.status = ? AND workflow_approvals.level = 2 AND approval_workflows.current_level = 2",
			models.WorkflowStatusPending, models.WorkflowStatusPending).
		Count(&count)
	stats["techPending"] = count

	s.db.Model(&models.WfNode{}).
		Joins("JOIN approval_workflows ON approval_workflows.id = workflow_approvals.workflow_id").
		Where("workflow_approvals.status = ? AND approval_workflows.status = ? AND workflow_approvals.level = 3 AND approval_workflows.current_level = 3",
			models.WorkflowStatusPending, models.WorkflowStatusPending).
		Count(&count)
	stats["financePending"] = count

	// 待审批总数
	s.db.Model(&models.WfNode{}).
		Joins("JOIN approval_workflows ON approval_workflows.id = workflow_approvals.workflow_id").
		Where("workflow_approvals.status = ? AND approval_workflows.status = ?",
			models.WorkflowStatusPending, models.WorkflowStatusPending).
		Count(&count)
	stats["pendingCount"] = count

	// 被拒绝的合同数量（工作流状态为 rejected）
	s.db.Model(&models.ApprovalWorkflow{}).
		Where("status = ?", "rejected").
		Count(&count)
	stats["rejectedCount"] = count

	// 已生效合同数量
	s.db.Model(&models.Contract{}).
		Where("status = ?", models.StatusActive).
		Count(&count)
	stats["activeCount"] = count

	// 已完成合同数量
	s.db.Model(&models.Contract{}).
		Where("status = ?", models.StatusCompleted).
		Count(&count)
	stats["completedCount"] = count

	return stats, nil
}
