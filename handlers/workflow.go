package handlers

import (
	"contract-manage/middleware"
	"contract-manage/models"
	"contract-manage/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WorkflowHandler 审批工作流处理器
// 处理合同审批工作流的创建、审批、查询等操作
type WorkflowHandler struct {
	workflowService *services.WorkflowService // 工作流服务实例
}

// NewWorkflowHandler 创建工作流处理器实例
// 参数：db-数据库连接实例
// 返回：配置好的WorkflowHandler指针
func NewWorkflowHandler(db *gorm.DB) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: services.NewWorkflowService(db),
	}
}

// GetWorkflow 获取工作流处理器
// 根据合同ID获取该合同对应的审批工作流
// GET /api/workflow/:contract_id
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	// 解析合同ID参数
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	// 调用服务层获取工作流信息
	workflow, err := h.workflowService.GetWorkflowByContractID(contractID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CreateWorkflow 创建工作流处理器
// 为合同创建新的审批工作流
// POST /api/workflow/create
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	// 定义请求结构
	var input struct {
		ContractID  uint64 `json:"contract_id" binding:"required"`  // 合同ID，必填
		CreatorRole string `json:"creator_role" binding:"required"` // 创建者角色，必填
	}

	// 解析请求体
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取创建人ID
	userID, _ := middleware.GetCurrentUserID(c)

	// 调用服务层创建工作流
	workflow, err := h.workflowService.CreateWorkflow(input.ContractID, uint64(userID), input.CreatorRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow"})
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// Approve 审批通过处理器
// 审批人同意当前审批节点，工作流进入下一级
// POST /api/workflow/approve
func (h *WorkflowHandler) Approve(c *gin.Context) {
	// 定义请求结构
	var input struct {
		WorkflowID uint64 `json:"workflow_id" binding:"required"` // 工作流ID，必填
		Level      int    `json:"level" binding:"required"`       // 审批级别，必填
		Comment    string `json:"comment"`                        // 审批意见，可选
	}

	// 解析请求体
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文获取当前登录用户信息
	userID, _ := middleware.GetCurrentUserID(c)
	role, _ := middleware.GetCurrentUserRole(c)

	// 从工作流获取合同ID
	var workflow models.ApprovalWorkflow
	if err := h.workflowService.GetWorkflowByWorkflowID(input.WorkflowID, &workflow); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作流不存在"})
		return
	}

	// 调用服务层执行审批通过操作
	err := h.workflowService.Approve(input.WorkflowID, workflow.ContractID, input.Level, uint64(userID), role, input.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "审批通过成功"})
}

// Reject 审批拒绝处理器
// 审批人拒绝当前审批节点，整个工作流结束
// POST /api/workflow/reject
func (h *WorkflowHandler) Reject(c *gin.Context) {
	// 定义请求结构
	var input struct {
		WorkflowID uint64 `json:"workflow_id" binding:"required"` // 工作流ID，必填
		Level      int    `json:"level" binding:"required"`       // 审批级别，必填
		Comment    string `json:"comment"`                        // 拒绝理由
	}

	// 解析请求体
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文获取当前登录用户信息
	userID, _ := middleware.GetCurrentUserID(c)
	role, _ := middleware.GetCurrentUserRole(c)

	// 调用服务层执行审批拒绝操作
	err := h.workflowService.Reject(input.WorkflowID, input.Level, uint64(userID), role, input.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "审批已拒绝"})
}

// GetMyPendingApproval 获取我的待审批列表处理器
// 返回当前用户需要审批的所有工作流
// GET /api/workflow/:contract_id/pending
func (h *WorkflowHandler) GetMyPendingApproval(c *gin.Context) {
	role, _ := middleware.GetCurrentUserRole(c)
	results, err := h.workflowService.GetPendingApprovals(role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending approvals"})
		return
	}
	c.JSON(http.StatusOK, results)
}

// GetWorkflowStatus 获取合同的工作流状态
// GET /api/workflow/:contract_id/status
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	status, err := h.workflowService.GetWorkflowStatus(contractID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// SendApprovalReminder 发送审批提醒处理器
// POST /api/workflow/:contract_id/remind
func (h *WorkflowHandler) SendApprovalReminder(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contract_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contract ID"})
		return
	}

	userID, _ := middleware.GetCurrentUserID(c)
	if err := h.workflowService.SendApprovalReminder(contractID, uint64(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "提醒已发送"})
}

// GetMyNotifications 获取我的通知列表
// GET /api/notifications
func (h *WorkflowHandler) GetMyNotifications(c *gin.Context) {
	userID, _ := middleware.GetCurrentUserID(c)
	notifications, err := h.workflowService.GetUserNotifications(uint64(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}
	c.JSON(http.StatusOK, notifications)
}

// MarkNotificationRead 标记通知已读
// PUT /api/notifications/:id/read
func (h *WorkflowHandler) MarkNotificationRead(c *gin.Context) {
	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	userID, _ := middleware.GetCurrentUserID(c)
	if err := h.workflowService.MarkNotificationRead(notificationID, uint64(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已标记为已读"})
}

// GetUnreadNotificationCount 获取未读通知数量
// GET /api/notifications/unread-count
func (h *WorkflowHandler) GetUnreadNotificationCount(c *gin.Context) {
	userID, _ := middleware.GetCurrentUserID(c)
	count, err := h.workflowService.GetUnreadNotificationCount(uint64(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get count"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetApprovalStats 获取审批统计数据
// GET /api/workflow/stats
func (h *WorkflowHandler) GetApprovalStats(c *gin.Context) {
	stats, err := h.workflowService.GetApprovalStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// DeleteNotification 删除通知
// DELETE /api/notifications/:id
func (h *WorkflowHandler) DeleteNotification(c *gin.Context) {
	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	userID, _ := middleware.GetCurrentUserID(c)
	if err := h.workflowService.DeleteNotification(notificationID, uint64(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// DeleteAllNotifications 删除当前用户所有通知
// DELETE /api/notifications/all
func (h *WorkflowHandler) DeleteAllNotifications(c *gin.Context) {
	userID, _ := middleware.GetCurrentUserID(c)
	if err := h.workflowService.DeleteAllNotifications(uint64(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All notifications deleted"})
}
