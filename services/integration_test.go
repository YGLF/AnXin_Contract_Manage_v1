package services

import (
	"log"
	"testing"
	"time"

	"contract-manage/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

func setupIntegrationTestDB(t *testing.T) func() {
	dsn := "root:rootroots@tcp(192.168.31.20:3306)/contract_manage?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	models.DB = db

	testDB = db

	return func() {
		sqlDB.Close()
	}
}

func cleanupTestData(db *gorm.DB) {
	db.Exec("DELETE FROM notifications WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'test_%')")
	db.Exec("DELETE FROM reminders WHERE contract_id IN (SELECT id FROM contracts WHERE contract_no LIKE 'TEST-%')")
	db.Exec("DELETE FROM contract_lifecycle_events WHERE contract_id IN (SELECT id FROM contracts WHERE contract_no LIKE 'TEST-%')")
	db.Exec("DELETE FROM workflow_approvals WHERE workflow_id IN (SELECT id FROM approval_workflows WHERE contract_id IN (SELECT id FROM contracts WHERE contract_no LIKE 'TEST-%'))")
	db.Exec("DELETE FROM approval_workflows WHERE contract_id IN (SELECT id FROM contracts WHERE contract_no LIKE 'TEST-%')")
	db.Exec("DELETE FROM documents WHERE contract_id IN (SELECT id FROM contracts WHERE contract_no LIKE 'TEST-%')")
	db.Exec("DELETE FROM contracts WHERE contract_no LIKE 'TEST-%'")
	db.Exec("DELETE FROM customers WHERE code LIKE 'TEST-%'")
	db.Exec("DELETE FROM contract_types WHERE code LIKE 'TEST-%'")
	db.Exec("DELETE FROM users WHERE username LIKE 'test_%'")
}

func getOrCreateUser(db *gorm.DB, username, email, password, role string) (*models.User, error) {
	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err == nil {
		return &user, nil
	}

	hashedPassword := password
	user = models.User{
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
		FullName:       username,
		Role:           models.UserRole(role),
		IsActive:       true,
	}

	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func TestFullApprovalWorkflow(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：完整审批流程 ===")

	cleanupTestData(db)

	log.Println("[1] 创建测试用户...")
	salesUser, err := getOrCreateUser(db, "test_sales", "test_sales@example.com", "password123", "sales")
	if err != nil {
		t.Fatalf("创建销售人员失败: %v", err)
	}
	log.Printf("    销售人员: %s (ID: %d)", salesUser.Username, salesUser.ID)

	salesDirectorUser, err := getOrCreateUser(db, "test_sales_director", "test_sales_director@example.com", "password123", "sales_director")
	if err != nil {
		t.Fatalf("创建销售总监失败: %v", err)
	}
	log.Printf("    销售总监: %s (ID: %d)", salesDirectorUser.Username, salesDirectorUser.ID)

	techDirectorUser, err := getOrCreateUser(db, "test_tech_director", "test_tech_director@example.com", "password123", "tech_director")
	if err != nil {
		t.Fatalf("创建技术总监失败: %v", err)
	}
	log.Printf("    技术总监: %s (ID: %d)", techDirectorUser.Username, techDirectorUser.ID)

	financeDirectorUser, err := getOrCreateUser(db, "test_finance_director", "test_finance_director@example.com", "password123", "finance_director")
	if err != nil {
		t.Fatalf("创建财务总监失败: %v", err)
	}
	log.Printf("    财务总监: %s (ID: %d)", financeDirectorUser.Username, financeDirectorUser.ID)

	log.Println("[2] 销售人员创建客户...")
	customer := models.Customer{
		Name:          "测试客户A",
		Code:          "TEST-C001",
		Type:          "customer",
		ContactPerson: "张三",
		ContactPhone:  "13800138000",
		IsActive:      true,
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("创建客户失败: %v", err)
	}
	log.Printf("    客户: %s (ID: %d)", customer.Name, customer.ID)

	log.Println("[3] 销售人员创建合同类型...")
	contractType := models.ContractType{
		Name:        "测试采购合同",
		Code:        "TEST-PO001",
		Description: "测试用采购合同类型",
	}
	if err := db.Create(&contractType).Error; err != nil {
		t.Fatalf("创建合同类型失败: %v", err)
	}
	log.Printf("    合同类型: %s (ID: %d)", contractType.Name, contractType.ID)

	log.Println("[4] 销售人员创建合同...")
	signDate := time.Now()
	startDate := time.Now()
	endDate := time.Now().AddDate(1, 0, 0)

	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0001",
		Title:          "测试采购合同A项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         100000.00,
		Currency:       "CNY",
		Status:         models.StatusDraft,
		SignDate:       &signDate,
		StartDate:      &startDate,
		EndDate:        &endDate,
		PaymentTerms:   "预付30%",
		CreatorID:      salesUser.ID,
	}
	if err := db.Create(&contract).Error; err != nil {
		t.Fatalf("创建合同失败: %v", err)
	}
	log.Printf("    合同: %s (ID: %d, Status: %s)", contract.ContractNo, contract.ID, contract.Status)

	log.Println("[5] 销售人员提交审批...")
	workflowService := NewWorkflowService(db)
	workflow, err := workflowService.CreateWorkflow(uint64(contract.ID), uint64(salesUser.ID), string(models.RoleSales))
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	log.Printf("    工作流: ID=%d, CurrentLevel=%d, MaxLevel=%d", workflow.ID, workflow.CurrentLevel, workflow.MaxLevel)

	var updatedContract models.Contract
	if err := db.First(&updatedContract, contract.ID).Error; err != nil {
		t.Fatalf("获取合同失败: %v", err)
	}
	if updatedContract.Status != models.StatusPending {
		t.Errorf("提交审批后合同状态应为 pending, got %s", updatedContract.Status)
	}
	log.Printf("    合同状态已变更为: %s", updatedContract.Status)

	log.Println("[6] 销售总监审批 (同意)...")
	if err := workflowService.Approve(workflow.ID, uint64(contract.ID), 1, uint64(salesDirectorUser.ID), string(models.RoleSalesDirector), "同意，方案合理"); err != nil {
		t.Fatalf("销售总监审批失败: %v", err)
	}

	var wf models.ApprovalWorkflow
	if err := db.First(&wf, workflow.ID).Error; err != nil {
		t.Fatalf("获取工作流失败: %v", err)
	}
	if wf.CurrentLevel != 2 {
		t.Errorf("销售总监审批后工作流级别应为 2, got %d", wf.CurrentLevel)
	}
	log.Printf("    工作流流转到级别: %d (技术总监审批)", wf.CurrentLevel)

	log.Println("[7] 技术总监审批 (同意)...")
	if err := workflowService.Approve(workflow.ID, uint64(contract.ID), 2, uint64(techDirectorUser.ID), string(models.RoleTechDirector), "同意，技术方案可行"); err != nil {
		t.Fatalf("技术总监审批失败: %v", err)
	}

	if err := db.First(&wf, workflow.ID).Error; err != nil {
		t.Fatalf("获取工作流失败: %v", err)
	}
	if wf.CurrentLevel != 3 {
		t.Errorf("技术总监审批后工作流级别应为 3, got %d", wf.CurrentLevel)
	}
	log.Printf("    工作流流转到级别: %d (财务总监审批)", wf.CurrentLevel)

	log.Println("[8] 财务总监审批 (同意)...")
	if err := workflowService.Approve(workflow.ID, uint64(contract.ID), 3, uint64(financeDirectorUser.ID), string(models.RoleFinanceDirector), "同意，预算合理"); err != nil {
		t.Fatalf("财务总监审批失败: %v", err)
	}

	if err := db.First(&wf, workflow.ID).Error; err != nil {
		t.Fatalf("获取工作流失败: %v", err)
	}
	if wf.Status != models.WorkflowStatusCompleted {
		t.Errorf("财务总监审批后工作流状态应为 completed, got %s", wf.Status)
	}

	if err := db.First(&updatedContract, contract.ID).Error; err != nil {
		t.Fatalf("获取合同失败: %v", err)
	}
	if updatedContract.Status != models.StatusInProgress {
		t.Errorf("审批全部通过后合同状态应为 in_progress, got %s", updatedContract.Status)
	}
	log.Printf("    合同最终状态: %s", updatedContract.Status)
	log.Printf("    工作流最终状态: %s", wf.Status)

	log.Println("=== 测试通过: 完整审批流程（全部同意）===")
}

func TestApprovalWorkflow_SalesRejected(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：销售总监拒绝 ===")

	cleanupTestData(db)

	salesUser, _ := getOrCreateUser(db, "test_sales2", "test_sales2@example.com", "password123", "sales")
	salesDirectorUser, _ := getOrCreateUser(db, "test_sales_director2", "test_sales_director2@example.com", "password123", "sales_director")

	customer := models.Customer{Name: "测试客户B", Code: "TEST-C002", Type: "customer", IsActive: true}
	db.Create(&customer)

	contractType := models.ContractType{Name: "测试服务合同", Code: "TEST-SV001"}
	db.Create(&contractType)

	signDate := time.Now()
	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0002",
		Title:          "测试服务合同B项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         50000.00,
		Currency:       "CNY",
		Status:         models.StatusDraft,
		SignDate:       &signDate,
		CreatorID:      salesUser.ID,
	}
	db.Create(&contract)

	workflowService := NewWorkflowService(db)
	workflow, _ := workflowService.CreateWorkflow(uint64(contract.ID), uint64(salesUser.ID), string(models.RoleSales))

	log.Println("[销售总监审批 - 拒绝]")
	err := workflowService.Reject(workflow.ID, 1, uint64(salesDirectorUser.ID), string(models.RoleSalesDirector), "方案需要修改")
	if err != nil {
		t.Fatalf("销售总监拒绝失败: %v", err)
	}

	var updatedContract models.Contract
	db.First(&updatedContract, contract.ID)
	if updatedContract.Status != models.StatusDraft {
		t.Errorf("被拒绝后合同状态应为 draft, got %s", updatedContract.Status)
	}

	var wf models.ApprovalWorkflow
	db.First(&wf, workflow.ID)
	if wf.CurrentLevel != 1 {
		t.Errorf("被拒绝后工作流级别应重置为 1, got %d", wf.CurrentLevel)
	}

	log.Printf("    合同状态: %s (应为 draft)", updatedContract.Status)
	log.Printf("    工作流级别: %d (应重置为 1)", wf.CurrentLevel)
	log.Println("=== 测试通过: 销售总监拒绝流程 ===")
}

func TestApprovalWorkflow_TechRejected(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：技术总监拒绝 ===")

	cleanupTestData(db)

	salesUser, _ := getOrCreateUser(db, "test_sales3", "test_sales3@example.com", "password123", "sales")
	salesDirectorUser, _ := getOrCreateUser(db, "test_sales_director3", "test_sales_director3@example.com", "password123", "sales_director")
	techDirectorUser, _ := getOrCreateUser(db, "test_tech_director3", "test_tech_director3@example.com", "password123", "tech_director")

	customer := models.Customer{Name: "测试客户C", Code: "TEST-C003", Type: "customer", IsActive: true}
	db.Create(&customer)

	contractType := models.ContractType{Name: "测试技术合同", Code: "TEST-TECH001"}
	db.Create(&contractType)

	signDate := time.Now()
	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0003",
		Title:          "测试技术合同C项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         80000.00,
		Currency:       "CNY",
		Status:         models.StatusDraft,
		SignDate:       &signDate,
		CreatorID:      salesUser.ID,
	}
	db.Create(&contract)

	workflowService := NewWorkflowService(db)
	workflow, _ := workflowService.CreateWorkflow(uint64(contract.ID), uint64(salesUser.ID), string(models.RoleSales))

	workflowService.Approve(workflow.ID, uint64(contract.ID), 1, uint64(salesDirectorUser.ID), string(models.RoleSalesDirector), "同意")

	log.Println("[技术总监审批 - 拒绝]")
	err := workflowService.Reject(workflow.ID, 2, uint64(techDirectorUser.ID), string(models.RoleTechDirector), "技术方案不可行")
	if err != nil {
		t.Fatalf("技术总监拒绝失败: %v", err)
	}

	var updatedContract models.Contract
	db.First(&updatedContract, contract.ID)
	if updatedContract.Status != models.StatusDraft {
		t.Errorf("被拒绝后合同状态应为 draft, got %s", updatedContract.Status)
	}

	var wf models.ApprovalWorkflow
	db.First(&wf, workflow.ID)
	if wf.CurrentLevel != 1 {
		t.Errorf("被拒绝后工作流级别应重置为 1, got %d", wf.CurrentLevel)
	}

	log.Printf("    合同状态: %s (应为 draft)", updatedContract.Status)
	log.Printf("    工作流级别: %d (应重置为 1)", wf.CurrentLevel)
	log.Println("=== 测试通过: 技术总监拒绝流程 ===")
}

func TestApprovalWorkflow_FinanceRejected(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：财务总监拒绝 ===")

	cleanupTestData(db)

	salesUser, _ := getOrCreateUser(db, "test_sales4", "test_sales4@example.com", "password123", "sales")
	salesDirectorUser, _ := getOrCreateUser(db, "test_sales_director4", "test_sales_director4@example.com", "password123", "sales_director")
	techDirectorUser, _ := getOrCreateUser(db, "test_tech_director4", "test_tech_director4@example.com", "password123", "tech_director")
	financeDirectorUser, _ := getOrCreateUser(db, "test_finance_director4", "test_finance_director4@example.com", "password123", "finance_director")

	customer := models.Customer{Name: "测试客户D", Code: "TEST-C004", Type: "customer", IsActive: true}
	db.Create(&customer)

	contractType := models.ContractType{Name: "测试财务合同", Code: "TEST-FIN001"}
	db.Create(&contractType)

	signDate := time.Now()
	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0004",
		Title:          "测试财务合同D项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         120000.00,
		Currency:       "CNY",
		Status:         models.StatusDraft,
		SignDate:       &signDate,
		CreatorID:      salesUser.ID,
	}
	db.Create(&contract)

	workflowService := NewWorkflowService(db)
	workflow, _ := workflowService.CreateWorkflow(uint64(contract.ID), uint64(salesUser.ID), string(models.RoleSales))

	workflowService.Approve(workflow.ID, uint64(contract.ID), 1, uint64(salesDirectorUser.ID), string(models.RoleSalesDirector), "同意")
	workflowService.Approve(workflow.ID, uint64(contract.ID), 2, uint64(techDirectorUser.ID), string(models.RoleTechDirector), "同意")

	log.Println("[财务总监审批 - 拒绝]")
	err := workflowService.Reject(workflow.ID, 3, uint64(financeDirectorUser.ID), string(models.RoleFinanceDirector), "预算超支")
	if err != nil {
		t.Fatalf("财务总监拒绝失败: %v", err)
	}

	var updatedContract models.Contract
	db.First(&updatedContract, contract.ID)
	if updatedContract.Status != models.StatusDraft {
		t.Errorf("被拒绝后合同状态应为 draft, got %s", updatedContract.Status)
	}

	var wf models.ApprovalWorkflow
	db.First(&wf, workflow.ID)
	if wf.CurrentLevel != 1 {
		t.Errorf("被拒绝后工作流级别应重置为 1, got %d", wf.CurrentLevel)
	}

	log.Printf("    合同状态: %s (应为 draft)", updatedContract.Status)
	log.Printf("    工作流级别: %d (应重置为 1)", wf.CurrentLevel)
	log.Println("=== 测试通过: 财务总监拒绝流程 ===")
}

func TestContractExpiryReminder(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：合同到期提醒 ===")

	cleanupTestData(db)

	salesUser, _ := getOrCreateUser(db, "test_sales5", "test_sales5@example.com", "password123", "sales")

	customer := models.Customer{Name: "测试客户E", Code: "TEST-C005", Type: "customer", IsActive: true}
	db.Create(&customer)

	contractType := models.ContractType{Name: "测试合同E", Code: "TEST-E001"}
	db.Create(&contractType)

	signDate := time.Now()
	startDate := time.Now()
	expiryDate := time.Now().AddDate(0, 0, 30)

	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0005",
		Title:          "测试到期提醒合同E项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         60000.00,
		Currency:       "CNY",
		Status:         models.StatusActive,
		SignDate:       &signDate,
		StartDate:      &startDate,
		EndDate:        &expiryDate,
		CreatorID:      salesUser.ID,
	}
	db.Create(&contract)

	log.Printf("    合同: %s", contract.ContractNo)
	log.Printf("    到期日期: %s", expiryDate.Format("2006-01-02"))

	approvalService := NewApprovalService()
	reminderDate := time.Now().AddDate(0, 0, 7)
	reminder, err := approvalService.CreateReminder(ReminderCreateInput{
		ContractID:   contract.ID,
		Type:         "expiry",
		ReminderDate: &JSONTime{Time: reminderDate},
		DaysBefore:   7,
	})
	if err != nil {
		t.Fatalf("创建到期提醒失败: %v", err)
	}

	log.Printf("    提醒创建成功: ID=%d, 提醒日期=%s, 提前天数=%d", reminder.ID, reminderDate.Format("2006-01-02"), reminder.DaysBefore)

	reminders, err := approvalService.GetReminders(contract.ID)
	if err != nil {
		t.Fatalf("获取提醒列表失败: %v", err)
	}
	if len(reminders) != 1 {
		t.Errorf("应存在1条提醒, got %d", len(reminders))
	}

	log.Println("=== 测试通过: 合同到期提醒 ===")
}

func TestApprovalReminderNotification(t *testing.T) {
	cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	db := testDB

	log.Println("=== 开始集成测试：审批催办通知 ===")

	cleanupTestData(db)

	salesUser, _ := getOrCreateUser(db, "test_sales6", "test_sales6@example.com", "password123", "sales")
	salesDirectorUser, _ := getOrCreateUser(db, "test_sales_director6", "test_sales_director6@example.com", "password123", "sales_director")

	customer := models.Customer{Name: "测试客户F", Code: "TEST-C006", Type: "customer", IsActive: true}
	db.Create(&customer)

	contractType := models.ContractType{Name: "测试合同F", Code: "TEST-F001"}
	db.Create(&contractType)

	signDate := time.Now()
	contract := models.Contract{
		ContractNo:     "TEST-CT-2024-0006",
		Title:          "测试催办通知合同F项目",
		CustomerID:     customer.ID,
		ContractTypeID: contractType.ID,
		Amount:         70000.00,
		Currency:       "CNY",
		Status:         models.StatusDraft,
		SignDate:       &signDate,
		CreatorID:      salesUser.ID,
	}
	db.Create(&contract)

	workflowService := NewWorkflowService(db)
	workflow, _ := workflowService.CreateWorkflow(uint64(contract.ID), uint64(salesUser.ID), string(models.RoleSales))

	workflowService.Approve(workflow.ID, uint64(contract.ID), 1, uint64(salesDirectorUser.ID), string(models.RoleSalesDirector), "同意")

	log.Println("[销售人员催办]")
	err := workflowService.SendApprovalReminder(uint64(contract.ID), uint64(salesUser.ID))
	if err != nil {
		t.Fatalf("发送催办通知失败: %v", err)
	}

	var notifications []models.Notification
	db.Where("target_role = ? AND notification_type = ?", "tech_director", models.NotificationTypeApprovalReminder).Find(&notifications)

	log.Printf("    待审批人(技术总监)收到通知数量: %d", len(notifications))
	if len(notifications) == 0 {
		t.Error("待审批人应收到催办通知")
	}

	if len(notifications) > 0 {
		log.Printf("    通知内容: %s", notifications[0].Content)
	}

	log.Println("=== 测试通过: 审批催办通知 ===")
}
