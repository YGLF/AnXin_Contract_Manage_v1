package routes

import (
	"contract-manage/handlers"
	"contract-manage/middleware"
	"contract-manage/models"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	contractHandler := handlers.NewContractHandler(models.DB)
	approvalHandler := handlers.NewApprovalHandler()
	workflowHandler := handlers.NewWorkflowHandler(models.DB)
	auditHandler := handlers.NewAuditHandler()
	customerHandler := handlers.NewCustomerHandler()
	authHandler := handlers.NewAuthHandler()
	cryptoHandler := handlers.NewCryptoHandler()

	// 加密服务路由（仅管理员可配置）
	admin := r.Group("/api/crypto")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.RequirePermission("user.manage"))
	{
		admin.POST("/config-hsm", cryptoHandler.ConfigHSM)
		admin.POST("/config-sm4", cryptoHandler.ConfigSM4)
		admin.POST("/config-aes", cryptoHandler.ConfigAES)
		admin.POST("/generate-key", cryptoHandler.GenerateKey)
		admin.GET("/status", cryptoHandler.GetCryptoStatus)
	}

	// 加解密路由（需要加密权限）
	crypto := r.Group("/api/crypto")
	crypto.Use(middleware.AuthMiddleware())
	{
		crypto.POST("/encrypt", cryptoHandler.Encrypt)
		crypto.POST("/decrypt", cryptoHandler.Decrypt)
	}

	auth := r.Group("/api/auth")
	auth.Use(middleware.RateLimitMiddleware())
	{
		auth.POST("/register", middleware.RegisterRateLimitMiddleware(), authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", middleware.AuthMiddleware(), authHandler.RefreshToken)
		auth.GET("/me", middleware.AuthMiddleware(), authHandler.GetCurrentUserInfo)
		auth.GET("/users", middleware.AuthMiddleware(), middleware.RequirePermission("user.manage"), authHandler.GetUsers)
		auth.GET("/users/:user_id", middleware.AuthMiddleware(), middleware.RequirePermission("user.manage"), authHandler.GetUserByID)
		auth.PUT("/users/:user_id", middleware.AuthMiddleware(), middleware.RequirePermission("user.manage"), authHandler.UpdateUser)
		auth.DELETE("/users/:user_id", middleware.AuthMiddleware(), middleware.RequirePermission("user.manage"), authHandler.DeleteUser)
		auth.POST("/users/:user_id/reset-password", middleware.AuthMiddleware(), authHandler.ResetPassword)
		auth.POST("/users/:user_id/unlock", middleware.AuthMiddleware(), authHandler.UnlockUser)
		auth.GET("/users/:user_id/lock-status", middleware.AuthMiddleware(), authHandler.GetUserLockStatus)
	}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	api.Use(handlers.AuditLogMiddleware(handlers.GetAuditService()))
	{
		api.GET("/customers", middleware.RequirePermission("customer.read"), customerHandler.GetCustomers)
		api.GET("/customers/:customer_id", middleware.RequirePermission("customer.read"), customerHandler.GetCustomerByID)
		api.POST("/customers", middleware.RequirePermission("customer.create"), customerHandler.CreateCustomer)
		api.PUT("/customers/:customer_id", middleware.RequirePermission("customer.edit"), customerHandler.UpdateCustomer)
		api.DELETE("/customers/:customer_id", middleware.RequirePermission("customer.delete"), customerHandler.DeleteCustomer)

		api.GET("/contract-types", middleware.RequirePermission("customer.read"), customerHandler.GetContractTypes)
		api.POST("/contract-types", middleware.RequirePermission("contract_type.manage"), customerHandler.CreateContractType)
		api.PUT("/contract-types/:type_id", middleware.RequirePermission("contract_type.manage"), customerHandler.UpdateContractType)
		api.DELETE("/contract-types/:type_id", middleware.RequirePermission("contract_type.manage"), customerHandler.DeleteContractType)

		api.GET("/contracts", middleware.RequirePermission("contract.read"), contractHandler.GetContracts)
		api.POST("/contracts", middleware.RequirePermission("contract.create"), contractHandler.CreateContract)
		api.GET("/contracts/:contract_id", middleware.RequirePermission("contract.read"), contractHandler.GetContractByID)
		api.PUT("/contracts/:contract_id", middleware.RequirePermission("contract.edit"), contractHandler.UpdateContract)
		api.PUT("/contracts/:contract_id/status", middleware.RequirePermission("contract.edit"), contractHandler.UpdateContractStatus)
		api.POST("/contracts/:contract_id/status-change", middleware.RequirePermission("contract.edit"), contractHandler.CreateStatusChangeRequest)
		api.GET("/contracts/:contract_id/status-change", middleware.RequirePermission("contract.read"), contractHandler.GetStatusChangeRequests)
		api.POST("/contracts/:contract_id/archive", middleware.RequirePermission("contract.edit"), contractHandler.ArchiveContract)
		api.DELETE("/contracts/:contract_id", middleware.RequirePermission("contract.delete"), contractHandler.DeleteContract)
		api.GET("/contracts/:contract_id/lifecycle", middleware.RequirePermission("contract.read"), contractHandler.GetContractLifecycle)

		api.GET("/contracts/:contract_id/executions", middleware.RequirePermission("contract.read"), contractHandler.GetContractExecutions)
		api.POST("/contracts/:contract_id/executions", middleware.RequirePermission("contract.edit"), contractHandler.CreateContractExecution)
		api.DELETE("/executions/:execution_id", middleware.RequirePermission("contract.edit"), contractHandler.DeleteExecution)

		api.GET("/contracts/:contract_id/documents", middleware.RequirePermission("contract.read"), contractHandler.GetContractDocuments)
		api.POST("/contracts/:contract_id/documents", middleware.RequirePermission("contract.edit"), contractHandler.CreateContractDocument)
		api.GET("/documents/:document_id/preview", middleware.RequirePermission("contract.read"), contractHandler.PreviewDocument)
		api.DELETE("/documents/:document_id", middleware.RequirePermission("contract.delete"), contractHandler.DeleteDocument)

		api.GET("/contracts/:contract_id/approvals", middleware.RequirePermission("approval.view"), approvalHandler.GetContractApprovals)
		api.POST("/contracts/:contract_id/approvals", middleware.RequirePermission("contract.create"), approvalHandler.CreateApproval)
		api.PUT("/approvals/:approval_id", middleware.RequirePermission("approval.process"), approvalHandler.UpdateApproval)
		api.POST("/approvals/:approval_id/rollback", middleware.RequirePermission("approval.process"), approvalHandler.RollbackApproval)
		api.GET("/approvals/:approval_id/status", middleware.RequireAnyPermission("approval.process", "approval.view"), approvalHandler.GetApprovalStatus)
		api.POST("/approvals/process-expired", middleware.RequirePermission("approval.process"), approvalHandler.ProcessExpiredApprovals)
		api.GET("/pending-approvals", middleware.RequireAnyPermission("approval.process", "approval.view"), approvalHandler.GetPendingApprovals)

		api.POST("/workflow/create", middleware.RequirePermission("contract.create"), workflowHandler.CreateWorkflow)
		api.GET("/workflow/pending", middleware.RequireAnyPermission("approval.process", "approval.view"), workflowHandler.GetMyPendingApproval)
		api.GET("/workflow/:contract_id/status", middleware.RequirePermission("approval.view"), workflowHandler.GetWorkflowStatus)
		api.GET("/workflow/:contract_id", middleware.RequirePermission("approval.view"), workflowHandler.GetWorkflow)
		api.POST("/workflow/:contract_id/remind", middleware.RequirePermission("approval.view"), workflowHandler.SendApprovalReminder)
		api.POST("/workflow/approve", middleware.RequirePermission("approval.process"), workflowHandler.Approve)
		api.POST("/workflow/reject", middleware.RequirePermission("approval.process"), workflowHandler.Reject)

		api.GET("/notifications", middleware.RequireAnyPermission("approval.process", "approval.view", "dashboard"), workflowHandler.GetMyNotifications)
		api.PUT("/notifications/:id/read", middleware.RequireAnyPermission("approval.process", "approval.view", "dashboard"), workflowHandler.MarkNotificationRead)
		api.DELETE("/notifications/:id", middleware.RequireAnyPermission("approval.process", "approval.view", "dashboard"), workflowHandler.DeleteNotification)
		api.DELETE("/notifications/all", middleware.RequireAnyPermission("approval.process", "approval.view", "dashboard"), workflowHandler.DeleteAllNotifications)
		api.GET("/notifications/unread-count", middleware.RequireAnyPermission("approval.process", "approval.view", "dashboard"), workflowHandler.GetUnreadNotificationCount)

		api.GET("/pending-status-changes", middleware.RequirePermission("approval.process"), contractHandler.GetPendingStatusChangeApprovals)
		api.POST("/status-change-requests/:request_id/approve", middleware.RequirePermission("approval.process"), contractHandler.ApproveStatusChangeRequest)
		api.POST("/status-change-requests/:request_id/reject", middleware.RequirePermission("approval.process"), contractHandler.RejectStatusChangeRequest)

		api.GET("/contracts/:contract_id/reminders", middleware.RequirePermission("approval.view"), approvalHandler.GetContractReminders)
		api.POST("/contracts/:contract_id/reminders", middleware.RequirePermission("approval.process"), approvalHandler.CreateReminder)
		api.POST("/reminders/:reminder_id/send", middleware.RequirePermission("approval.process"), approvalHandler.SendReminder)

		api.GET("/expiring-contracts", middleware.RequirePermission("approval.view"), approvalHandler.GetExpiringContracts)
		api.GET("/statistics", middleware.RequirePermission("approval.view"), approvalHandler.GetStatistics)
		api.GET("/notifications/count", middleware.RequireAnyPermission("approval.process", "approval.view", "approval.view"), approvalHandler.GetNotificationCounts)

		api.GET("/audit-logs", middleware.RequirePermission("audit.view"), auditHandler.GetAuditLogs)
		api.DELETE("/audit-logs/:id", middleware.RequirePermission("user.manage"), auditHandler.DeleteAuditLog)
		api.POST("/audit-logs/batch-delete", middleware.RequirePermission("user.manage"), auditHandler.DeleteAuditLogs)
		api.GET("/audit-logs/export", middleware.RequirePermission("audit.view"), auditHandler.ExportAuditLogs)
	}
}
