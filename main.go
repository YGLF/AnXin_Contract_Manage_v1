package main

import (
	"contract-manage/config"
	"contract-manage/handlers"
	"contract-manage/middleware"
	"contract-manage/models"
	"fmt"
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	if err := models.InitDB(); err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	if err := models.InitAdmin(); err != nil {
		fmt.Println("Warning: Failed to create admin user: " + err.Error())
	}

	r := gin.Default()

	r.SetHTMLTemplate(template.Must(template.New("").Parse("")))

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(middleware.SecureHeadersMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())
	r.Use(middleware.RequestSizeLimitMiddleware(10 << 20))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "安心合同管理系统 API",
			"version": config.AppConfig.AppVersion,
			"time":    time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	authHandler := handlers.NewAuthHandler()
	customerHandler := handlers.NewCustomerHandler()
	contractHandler := handlers.NewContractHandler()
	approvalHandler := handlers.NewApprovalHandler()

	auth := r.Group("/api/auth")
	auth.Use(middleware.RateLimitMiddleware())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/users", middleware.AuthMiddleware(), authHandler.GetUsers)
		auth.GET("/users/:user_id", middleware.AuthMiddleware(), authHandler.GetUserByID)
		auth.PUT("/users/:user_id", middleware.AuthMiddleware(), authHandler.UpdateUser)
		auth.DELETE("/users/:user_id", middleware.AuthMiddleware(), authHandler.DeleteUser)
	}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		api.GET("/customers", customerHandler.GetCustomers)
		api.GET("/customers/:customer_id", customerHandler.GetCustomerByID)
		api.POST("/customers", customerHandler.CreateCustomer)
		api.PUT("/customers/:customer_id", customerHandler.UpdateCustomer)
		api.DELETE("/customers/:customer_id", customerHandler.DeleteCustomer)

		api.GET("/contract-types", customerHandler.GetContractTypes)
		api.POST("/contract-types", customerHandler.CreateContractType)

		api.GET("/contracts", contractHandler.GetContracts)
		api.GET("/contracts/:contract_id", contractHandler.GetContractByID)
		api.POST("/contracts", contractHandler.CreateContract)
		api.PUT("/contracts/:contract_id", contractHandler.UpdateContract)
		api.DELETE("/contracts/:contract_id", contractHandler.DeleteContract)

		api.GET("/contracts/:contract_id/executions", contractHandler.GetContractExecutions)
		api.POST("/contracts/:contract_id/executions", contractHandler.CreateContractExecution)

		api.GET("/contracts/:contract_id/documents", contractHandler.GetContractDocuments)
		api.POST("/contracts/:contract_id/documents", contractHandler.CreateContractDocument)
		api.DELETE("/documents/:document_id", contractHandler.DeleteDocument)

		api.GET("/contracts/:contract_id/approvals", approvalHandler.GetContractApprovals)
		api.POST("/contracts/:contract_id/approvals", approvalHandler.CreateApproval)

		api.PUT("/approvals/:approval_id", approvalHandler.UpdateApproval)

		api.GET("/contracts/:contract_id/reminders", approvalHandler.GetContractReminders)
		api.POST("/contracts/:contract_id/reminders", approvalHandler.CreateReminder)

		api.POST("/reminders/:reminder_id/send", approvalHandler.SendReminder)

		api.GET("/expiring-contracts", approvalHandler.GetExpiringContracts)
		api.GET("/statistics", approvalHandler.GetStatistics)
	}

	_ = approvalHandler
	_ = contractHandler

	addr := ":8000"
	fmt.Printf("服务器启动在 %s\n", addr)
	fmt.Printf("健康检查: http://localhost%s/health\n", addr)
	r.Run(addr)
}
