package handlers

import (
	"contract-manage/middleware"
	"contract-manage/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ApprovalHandler 审批处理器
// 处理合同审批、提醒、统计等相关请求
type ApprovalHandler struct {
	approvalService *services.ApprovalService // 审批服务实例
}

// NewApprovalHandler 创建审批处理器实例
// 返回：配置好的ApprovalHandler指针
func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{
		approvalService: services.NewApprovalService(),
	}
}

// GetContractApprovals 获取合同审批记录处理器
// 返回指定合同的所有审批历史记录
// GET /api/contracts/:contract_id/approvals
func (h *ApprovalHandler) GetContractApprovals(c *gin.Context) {
	// 解析合同ID参数
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	// 调用服务层获取审批记录
	approvals, err := h.approvalService.GetApprovalRecords(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvals)
}

// CreateApproval 创建审批记录处理器
// 为合同创建新的审批记录并提交审批
// POST /api/contracts/:contract_id/approvals
func (h *ApprovalHandler) CreateApproval(c *gin.Context) {
	// 解析合同ID参数
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	// 解析请求体
	var input services.ApprovalRecordCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前登录用户ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 获取当前用户角色
	role, _ := middleware.GetCurrentUserRole(c)
	if role == "" {
		role = "user"
	}

	// 设置合同ID和审批人角色
	input.ContractID = uint(contractID)
	input.ApproverRole = role

	// 调用服务层创建审批记录
	approval, err := h.approvalService.CreateApprovalRecord(input, userID, role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果是销售人员、用户、经理或管理员创建，提交审批
	if role == "sales" || role == "user" || role == "manager" || role == "admin" {
		h.approvalService.SubmitForApproval(uint(contractID), userID)
	}

	c.JSON(http.StatusCreated, approval)
}

// UpdateApproval 更新审批状态处理器
// 审批人对审批记录进行审批操作（通过/拒绝）
// PUT /api/approvals/:approval_id
func (h *ApprovalHandler) UpdateApproval(c *gin.Context) {
	// 解析审批记录ID参数
	id, err := strconv.ParseUint(c.Param("approval_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	// 解析请求体
	var input services.ApprovalRecordUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证审批记录是否存在
	record, _ := h.approvalService.GetApprovalRecordByID(uint(id))
	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval record not found"})
		return
	}

	// 获取当前用户角色
	role, _ := middleware.GetCurrentUserRole(c)
	if role == "" {
		role = "user"
	}

	// 传递操作人角色用于权限验证
	input.OperatorRole = role

	// 根据审批结果设置合同状态
	contractStatus := ""
	if input.Status == "approved" {
		contractStatus = "active" // 审批通过，合同生效
	} else if input.Status == "rejected" {
		contractStatus = "draft" // 审批拒绝，合同退回草稿
	}

	// 获取当前用户ID
	userID, _ := middleware.GetCurrentUserID(c)

	// 调用服务层更新审批记录
	approval, err := h.approvalService.UpdateApprovalRecord(uint(id), input, contractStatus, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approval)
}

// RollbackApproval 审批回退处理器
// 将已审批的记录回退到待审批状态
// POST /api/approvals/:approval_id/rollback
func (h *ApprovalHandler) RollbackApproval(c *gin.Context) {
	// 解析审批记录ID参数
	id, err := strconv.ParseUint(c.Param("approval_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	// 获取当前用户信息
	userID, _ := middleware.GetCurrentUserID(c)
	role, _ := middleware.GetCurrentUserRole(c)

	// 仅管理员可以回退审批
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以回退审批"})
		return
	}

	// 调用服务层回退审批
	approval, err := h.approvalService.RollbackApproval(uint(id), userID, role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "审批已回退",
		"approval": approval,
	})
}

// GetApprovalStatus 获取审批状态详情处理器
// 返回审批记录的详细信息和剩余时间
// GET /api/approvals/:approval_id/status
func (h *ApprovalHandler) GetApprovalStatus(c *gin.Context) {
	// 解析审批记录ID参数
	id, err := strconv.ParseUint(c.Param("approval_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	// 调用服务层获取状态信息
	statusInfo, err := h.approvalService.GetApprovalStatusInfo(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval record not found"})
		return
	}

	c.JSON(http.StatusOK, statusInfo)
}

// ProcessExpiredApprovals 处理超时审批处理器
// 自动将超时的审批标记为拒绝
// POST /api/approvals/process-expired
func (h *ApprovalHandler) ProcessExpiredApprovals(c *gin.Context) {
	// 仅管理员可以触发
	role, _ := middleware.GetCurrentUserRole(c)
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以执行此操作"})
		return
	}

	// 调用服务层处理超时审批
	count, err := h.approvalService.ProcessExpiredApprovals()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "已处理完成",
		"processed_count": count,
	})
}

// GetContractReminders 获取合同提醒列表处理器
// 返回指定合同的所有到期提醒设置
// GET /api/contracts/:contract_id/reminders
func (h *ApprovalHandler) GetContractReminders(c *gin.Context) {
	// 解析合同ID参数
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	// 调用服务层获取提醒列表
	reminders, err := h.approvalService.GetReminders(uint(contractID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reminders)
}

// CreateReminder 创建合同提醒处理器
// 为合同设置到期提醒
// POST /api/contracts/:contract_id/reminders
func (h *ApprovalHandler) CreateReminder(c *gin.Context) {
	// 解析合同ID参数
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	// 解析请求体
	var input services.ReminderCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置关联的合同ID
	input.ContractID = uint(contractID)

	// 调用服务层创建提醒
	reminder, err := h.approvalService.CreateReminder(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reminder)
}

// SendReminder 发送提醒处理器
// 手动触发提醒发送
// POST /api/reminders/:reminder_id/send
func (h *ApprovalHandler) SendReminder(c *gin.Context) {
	// 解析提醒ID参数
	id, err := strconv.ParseUint(c.Param("reminder_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reminder ID"})
		return
	}

	// 调用服务层标记提醒已发送
	if err := h.approvalService.UpdateReminderSent(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reminder not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reminder sent successfully"})
}

// GetExpiringContracts 获取即将到期合同处理器
// 返回指定天数内即将到期的合同列表
// GET /api/expiring-contracts
func (h *ApprovalHandler) GetExpiringContracts(c *gin.Context) {
	// 解析天数参数，默认30天
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	// 调用服务层获取即将到期合同
	contracts, err := h.approvalService.GetExpiringContracts(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contracts": contracts,
		"days":      days,
	})
}

// GetStatistics 获取统计数据处理器
// 返回合同管理的统计信息
// GET /api/statistics
func (h *ApprovalHandler) GetStatistics(c *gin.Context) {
	// 调用服务层获取统计数据
	stats, err := h.approvalService.GetStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPendingApprovals 获取待审批列表处理器
// 返回当前用户需要审批的审批记录
// GET /api/pending-approvals
func (h *ApprovalHandler) GetPendingApprovals(c *gin.Context) {
	// 获取当前用户ID和角色
	userID, _ := middleware.GetCurrentUserID(c)
	role, _ := middleware.GetCurrentUserRole(c)
	if role == "" {
		role = "user"
	}

	// 调用服务层获取待审批列表
	approvals, err := h.approvalService.GetPendingApprovalsByRole(role, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvals)
}

// GetNotificationCounts 获取通知数量处理器
// 返回各种待处理事项的数量，用于前端显示红点提示
// GET /api/notifications/count
func (h *ApprovalHandler) GetNotificationCounts(c *gin.Context) {
	// 获取当前用户角色
	role, _ := middleware.GetCurrentUserRole(c)
	if role == "" {
		role = "user"
	}

	// 初始化计数结果
	counts := map[string]int{}

	// 获取待审批数量
	pendingApprovals, _ := h.approvalService.GetPendingApprovalsByRole(role, 0)
	counts["pendingApprovals"] = len(pendingApprovals)

	// 如果是经理或管理员，获取待审批状态变更数量
	if role == "manager" || role == "admin" {
		pendingStatusChanges, _ := h.approvalService.GetPendingStatusChangesCount()
		counts["pendingStatusChanges"] = pendingStatusChanges
	} else {
		counts["pendingStatusChanges"] = 0
	}

	// 获取即将到期合同数量（30天内）
	expiringContracts, _ := h.approvalService.GetExpiringContracts(30)
	counts["expiringContracts"] = len(expiringContracts)

	// 计算总数
	counts["total"] = counts["pendingApprovals"] + counts["pendingStatusChanges"] + counts["expiringContracts"]

	c.JSON(http.StatusOK, counts)
}
