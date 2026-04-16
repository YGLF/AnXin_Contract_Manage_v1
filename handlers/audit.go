package handlers

import (
	"contract-manage/middleware"
	"contract-manage/models"
	"contract-manage/services"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuditHandler 审计日志处理器
// 处理审计日志的查询、删除、导出等操作
type AuditHandler struct {
	auditService *services.AuditService // 审计服务实例
}

// NewAuditHandler 创建审计处理器实例
// 返回：配置好的AuditHandler指针
func NewAuditHandler() *AuditHandler {
	return &AuditHandler{
		auditService: services.NewAuditService(),
	}
}

// GetAuditService 获取审计服务实例
// 用于中间件中创建审计日志
// 返回：审计服务指针
func GetAuditService() *services.AuditService {
	return services.NewAuditService()
}

// GetAuditLogs 获取审计日志列表处理器
// 支持分页和多条件筛选
// GET /api/audit-logs
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))           // 页码，默认第1页
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20")) // 每页数量，默认20条

	// 解析筛选条件
	username := c.Query("username")    // 按用户名筛选
	action := c.Query("action")        // 按操作类型筛选
	module := c.Query("module")        // 按模块筛选
	startDate := c.Query("start_date") // 开始日期
	endDate := c.Query("end_date")     // 结束日期
	statusCodeStr := c.Query("status_code")
	var statusCode *int
	if statusCodeStr != "" {
		if code, err := strconv.Atoi(statusCodeStr); err == nil {
			statusCode = &code
		}
	}

	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 调用服务层获取审计日志
	logs, total, err := h.auditService.GetAuditLogs(page, pageSize, username, action, module, startDate, endDate, statusCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回分页结果
	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,     // 日志列表
		"total": total,    // 总记录数
		"page":  page,     // 当前页码
		"size":  pageSize, // 每页数量
	})
}

// DeleteAuditLog 删除单条审计日志处理器
// 仅管理员和审计管理员可以删除
// DELETE /api/audit-logs/:id
func (h *AuditHandler) DeleteAuditLog(c *gin.Context) {
	// 权限检查：只有admin和audit_admin可以删除审计日志
	role, _ := middleware.GetCurrentUserRole(c)
	if role != "admin" && role != "audit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除审计日志，仅审计管理员可操作"})
		return
	}

	// 解析日志ID参数
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// 调用服务层删除日志
	if err := h.auditService.DeleteAuditLog(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// DeleteAuditLogs 批量删除审计日志处理器
// 仅管理员和审计管理员可以操作
// POST /api/audit-logs/batch-delete
func (h *AuditHandler) DeleteAuditLogs(c *gin.Context) {
	// 权限检查
	role, _ := middleware.GetCurrentUserRole(c)
	if role != "admin" && role != "audit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限删除审计日志，仅审计管理员可操作"})
		return
	}

	// 解析请求体，获取要删除的ID列表
	var input struct {
		IDs []uint `json:"ids"` // 要删除的日志ID数组
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 调用服务层批量删除
	if err := h.auditService.DeleteAuditLogs(input.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "批量删除成功"})
}

// ExportAuditLogs 导出审计日志处理器
// 根据筛选条件导出审计日志
// GET /api/audit-logs/export
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	// 解析筛选条件
	username := c.Query("username")
	action := c.Query("action")
	module := c.Query("module")
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	// 调用服务层获取日志（最多10000条）
	logs, _, err := h.auditService.GetAuditLogs(1, 10000, username, action, module, startDate, endDate, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// AuditLogMiddleware 审计日志中间件
// 在每个API请求处理后自动记录审计日志
// 记录请求的用户、操作类型、模块、IP地址等信息
// 返回：Gin中间件处理函数
func AuditLogMiddleware(auditService *services.AuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行后续处理函数
		c.Next()

		// 跳过OPTIONS预检请求
		if c.Request.Method == "OPTIONS" {
			return
		}

		// 跳过登录和注册请求（避免循环记录）
		if strings.HasPrefix(c.Request.URL.Path, "/api/auth/login") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/auth/register") {
			return
		}

		// 获取当前用户信息
		userID, _ := middleware.GetCurrentUserID(c)
		username, _ := c.Get("username")

		// 未登录用户不记录
		if userID == 0 {
			return
		}

		// 构建审计日志内容
		action := c.Request.Method + " " + c.Request.URL.Path // 操作：方法+路径
		module := getModuleFromPath(c.Request.URL.Path)       // 操作模块

		// 获取客户端信息
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 创建审计日志结构
		log := models.AuditLog{
			UserID:    userID,             // 用户ID
			Username:  username.(string),  // 用户名
			Action:    action,             // 操作描述
			Module:    module,             // 模块
			Method:    c.Request.Method,   // HTTP方法
			Path:      c.Request.URL.Path, // 请求路径
			IPAddress: clientIP,           // 客户端IP
			UserAgent: userAgent,          // 浏览器信息
		}

		// 记录响应状态码
		statusCode := c.Writer.Status()
		log.StatusCode = statusCode

		// 异步创建审计日志（不阻塞请求）
		go auditService.CreateAuditLog(log)
	}
}

// getModuleFromPath 从请求路径提取模块名称
// 用于审计日志分类
// 参数：path-请求URL路径
// 返回：模块名称
func getModuleFromPath(path string) string {
	if strings.Contains(path, "/auth/") {
		return "auth"
	}
	if strings.Contains(path, "/contracts") {
		return "contract"
	}
	if strings.Contains(path, "/customers") {
		return "customer"
	}
	if strings.Contains(path, "/approvals") || strings.Contains(path, "/pending") {
		return "approval"
	}
	if strings.Contains(path, "/reminders") {
		return "reminder"
	}
	if strings.Contains(path, "/users") {
		return "user"
	}
	if strings.Contains(path, "/statistics") {
		return "statistics"
	}
	return "other"
}
