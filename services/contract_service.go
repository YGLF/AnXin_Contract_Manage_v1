package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"contract-manage/models"

	"gorm.io/gorm"
)

// ContractService 合同服务
// 处理合同相关的业务逻辑
type ContractService struct{}

// NewContractService 创建合同服务实例
func NewContractService() *ContractService {
	return &ContractService{}
}

// JSONTime 自定义时间类型
// 支持多种JSON日期格式的解析
type JSONTime struct {
	time.Time
}

// UnmarshalJSON 实现json.Unmarshaler接口
// 支持解析ISO8601、RFC3339、Date三种格式
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		t.Time = time.Time{}
		return nil
	}

	formats := []string{
		"2006-01-02T15:04:05Z07:00", // ISO8601格式
		"2006-01-02T15:04:05",       // RFC3339格式
		"2006-01-02",                // 简单日期格式
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, str); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return errors.New("invalid date format")
}

// ContractCreateInput 合同创建输入结构
// 定义创建合同时需要的字段
type ContractCreateInput struct {
	Title          string    `json:"title" binding:"required"`            // 合同标题，必填
	CustomerID     uint      `json:"customer_id" binding:"required"`      // 客户ID，必填
	ContractTypeID uint      `json:"contract_type_id" binding:"required"` // 合同类型ID，必填
	Amount         float64   `json:"amount"`                              // 合同金额
	Currency       string    `json:"currency"`                            // 货币类型
	SignDate       *JSONTime `json:"sign_date"`                           // 签订日期
	StartDate      *JSONTime `json:"start_date"`                          // 开始日期
	EndDate        *JSONTime `json:"end_date"`                            // 结束日期
	PaymentTerms   string    `json:"payment_terms"`                       // 付款条款
	Content        string    `json:"content"`                             // 合同内容
	Notes          string    `json:"notes"`                               // 备注
}

// generateContractNo 生成合同编号
// 格式：CT+年月+4位序号，如CT2024010001
// 返回：生成的合同编号字符串
func (s *ContractService) generateContractNo() string {
	today := time.Now()
	prefix := fmt.Sprintf("CT%s", today.Format("200601"))

	var lastContract models.Contract
	models.DB.Where("contract_no LIKE ?", prefix+"%").Order("contract_no DESC").First(&lastContract)

	var newNo string
	if lastContract.ID != 0 {
		// 从已有合同编号中提取序号并递增
		lastNo := lastContract.ContractNo[len(lastContract.ContractNo)-4:]
		var num int
		fmt.Sscanf(lastNo, "%d", &num)
		newNo = fmt.Sprintf("%04d", num+1)
	} else {
		newNo = "0001"
	}

	return prefix + newNo
}

// GetContractByID 根据ID获取合同详情
// 参数：id-合同ID
// 返回：合同详情和错误信息
func (s *ContractService) GetContractByID(id uint) (*models.Contract, error) {
	var contract models.Contract
	if err := models.DB.Preload("Customer").Preload("Creator").Preload("ContractType").First(&contract, id).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

// GetContractByNo 根据合同编号获取合同
// 参数：contractNo-合同编号
// 返回：合同详情和错误信息
func (s *ContractService) GetContractByNo(contractNo string) (*models.Contract, error) {
	var contract models.Contract
	if err := models.DB.Where("contract_no = ?", contractNo).First(&contract).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

// ContractVisibilityParams 合同可见性参数
type ContractVisibilityParams struct {
	UserID uint   // 当前用户ID
	Role   string // 当前用户角色
}

// GetContracts 获取合同列表
// 支持分页、多条件筛选和角色可见性控制
// 可见性规则：
//   - 超级管理员/合同管理员：可查看所有合同
//   - 销售总监/技术总监/财务总监：只能查看需要自己审批的合同
//   - 销售人员：只能查看自己创建的合同
//
// 参数：skip-跳过记录数，limit-每页数量，customerID-客户ID筛选，contractTypeID-合同类型ID筛选，status-状态筛选，visibility-可见性参数
// 返回：合同列表和错误信息
func (s *ContractService) GetContracts(skip, limit int, customerID, contractTypeID uint, status, title string, visibility *ContractVisibilityParams) ([]models.Contract, error) {
	var contracts []models.Contract
	query := models.DB.Preload("Customer").Preload("Creator").Preload("ContractType")

	// 根据角色应用可见性规则
	if visibility != nil && visibility.Role != "" {
		switch models.UserRole(visibility.Role) {
		case models.RoleAdmin, models.RoleContractAdmin, models.RoleAuditAdmin:
			// 管理员和合同管理员可以查看所有合同，不添加额外过滤
		case models.RoleSalesDirector, models.RoleTechDirector, models.RoleFinanceDirector:
			// 总监只能看到需要自己审批的合同
			// 通过联表查询工作流审批记录
			query = query.Joins("JOIN approval_workflows ON approval_workflows.contract_id = contracts.id").
				Joins("JOIN workflow_approvals ON workflow_approvals.workflow_id = approval_workflows.id").
				Where("workflow_approvals.approver_role = ?", visibility.Role).
				Where("workflow_approvals.status = ?", "pending").
				Where("workflow_approvals.level = approval_workflows.current_level")
		case models.RoleSales:
			// 销售人员只能查看自己创建的合同
			query = query.Where("contracts.creator_id = ?", visibility.UserID)
		}
	}

	// 应用筛选条件
	if customerID > 0 {
		query = query.Where("customer_id = ?", customerID)
	}
	if contractTypeID > 0 {
		query = query.Where("contract_type_id = ?", contractTypeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}

	if err := query.Order("contracts.created_at DESC").Offset(skip).Limit(limit).Find(&contracts).Error; err != nil {
		return nil, err
	}

	// 手动加载提醒数据
	if len(contracts) > 0 {
		var contractIDs []uint
		for _, c := range contracts {
			contractIDs = append(contractIDs, c.ID)
		}
		var reminders []models.Reminder
		if err := models.DB.Where("contract_id IN ?", contractIDs).Find(&reminders).Error; err == nil {
			// 将提醒关联到对应的合同
			for i := range contracts {
				for _, r := range reminders {
					if r.ContractID == contracts[i].ID {
						contracts[i].Reminders = append(contracts[i].Reminders, r)
					}
				}
			}
		}
	}

	return contracts, nil
}

// GetContractsCount 获取合同总数（带可见性过滤）
func (s *ContractService) GetContractsCount(customerID, contractTypeID uint, status, title string, visibility *ContractVisibilityParams) (int64, error) {
	var count int64
	query := models.DB.Model(&models.Contract{})

	// 根据角色应用可见性规则
	if visibility != nil && visibility.Role != "" {
		switch models.UserRole(visibility.Role) {
		case models.RoleAdmin, models.RoleContractAdmin, models.RoleAuditAdmin:
			// 不添加过滤
		case models.RoleSalesDirector, models.RoleTechDirector, models.RoleFinanceDirector:
			query = query.Joins("JOIN approval_workflows ON approval_workflows.contract_id = contracts.id").
				Joins("JOIN workflow_approvals ON workflow_approvals.workflow_id = approval_workflows.id").
				Where("workflow_approvals.approver_role = ?", visibility.Role).
				Where("workflow_approvals.status = ?", "pending").
				Where("workflow_approvals.level = approval_workflows.current_level")
		case models.RoleSales:
			query = query.Where("creator_id = ?", visibility.UserID)
		}
	}

	if customerID > 0 {
		query = query.Where("customer_id = ?", customerID)
	}
	if contractTypeID > 0 {
		query = query.Where("contract_type_id = ?", contractTypeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CreateContract 创建合同
// 参数：input-合同创建输入，creatorID-创建人ID
// 返回：创建的合同和错误信息
func (s *ContractService) CreateContract(input ContractCreateInput, creatorID uint) (*models.Contract, error) {
	contract := models.Contract{
		ContractNo:     s.generateContractNo(), // 自动生成合同编号
		Title:          input.Title,
		CustomerID:     input.CustomerID,
		ContractTypeID: input.ContractTypeID,
		Amount:         input.Amount,
		Currency:       input.Currency,
		PaymentTerms:   input.PaymentTerms,
		Content:        input.Content,
		Notes:          input.Notes,
		CreatorID:      creatorID,
		Status:         models.StatusDraft, // 默认状态为草稿
	}

	// 设置日期字段
	if input.SignDate != nil && !input.SignDate.Time.IsZero() {
		contract.SignDate = &input.SignDate.Time
	}
	if input.StartDate != nil && !input.StartDate.Time.IsZero() {
		contract.StartDate = &input.StartDate.Time
	}
	if input.EndDate != nil && !input.EndDate.Time.IsZero() {
		contract.EndDate = &input.EndDate.Time
	}

	// 默认货币为人民币
	if contract.Currency == "" {
		contract.Currency = "CNY"
	}

	if err := models.DB.Create(&contract).Error; err != nil {
		return nil, err
	}

	// 添加生命周期事件
	s.AddLifecycleEvent(contract.ID, LifecycleEventInput{
		EventType:   string(models.LifecycleCreated),
		Description: "合同创建",
	}, creatorID)

	return &contract, nil
}

// ContractUpdateInput 合同更新输入结构
// 定义更新合同时可选的字段
type ContractUpdateInput struct {
	Title          string                `json:"title"`
	CustomerID     uint                  `json:"customer_id"`
	ContractTypeID uint                  `json:"contract_type_id"`
	Amount         float64               `json:"amount"`
	Currency       string                `json:"currency"`
	Status         models.ContractStatus `json:"status"`
	SignDate       *JSONTime             `json:"sign_date"`
	StartDate      *JSONTime             `json:"start_date"`
	EndDate        *JSONTime             `json:"end_date"`
	PaymentTerms   string                `json:"payment_terms"`
	Content        string                `json:"content"`
	Notes          string                `json:"notes"`
}

// UpdateContract 更新合同
// 参数：id-合同ID，input-更新输入
// 返回：更新后的合同和错误信息
func (s *ContractService) UpdateContract(id uint, input ContractUpdateInput) (*models.Contract, error) {
	contract, err := s.GetContractByID(id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	// 只更新有值的字段
	if input.Title != "" {
		updates["title"] = input.Title
	}
	if input.CustomerID > 0 {
		updates["customer_id"] = input.CustomerID
	}
	if input.ContractTypeID > 0 {
		updates["contract_type_id"] = input.ContractTypeID
	}
	if input.Amount > 0 {
		updates["amount"] = input.Amount
	}
	if input.Currency != "" {
		updates["currency"] = input.Currency
	}
	if input.Status != "" {
		updates["status"] = input.Status
	}
	if input.SignDate != nil && !input.SignDate.Time.IsZero() {
		updates["sign_date"] = input.SignDate.Time
	}
	if input.StartDate != nil && !input.StartDate.Time.IsZero() {
		updates["start_date"] = input.StartDate.Time
	}
	if input.EndDate != nil && !input.EndDate.Time.IsZero() {
		updates["end_date"] = input.EndDate.Time
	}
	if input.PaymentTerms != "" {
		updates["payment_terms"] = input.PaymentTerms
	}
	if input.Content != "" {
		updates["content"] = input.Content
	}
	if input.Notes != "" {
		updates["notes"] = input.Notes
	}

	if err := models.DB.Model(contract).Updates(updates).Error; err != nil {
		return nil, err
	}
	return contract, nil
}

// DeleteContract 删除合同
// 参数：id-合同ID
// 返回：错误信息
func (s *ContractService) DeleteContract(id uint) error {
	result := models.DB.Delete(&models.Contract{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// GetContractExecutionByID 根据ID获取执行记录
func (s *ContractService) GetContractExecutionByID(id uint) (*models.ContractExecution, error) {
	var execution models.ContractExecution
	if err := models.DB.First(&execution, id).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// GetContractExecutions 获取合同的执行记录列表
// 参数：contractID-合同ID
// 返回：执行记录列表和错误信息
func (s *ContractService) GetContractExecutions(contractID uint) ([]models.ContractExecution, error) {
	var executions []models.ContractExecution
	if err := models.DB.Where("contract_id = ?", contractID).Order("created_at DESC").Find(&executions).Error; err != nil {
		return nil, err
	}
	return executions, nil
}

// ContractExecutionCreateInput 执行记录创建输入结构
type ContractExecutionCreateInput struct {
	ContractID    uint      `json:"contract_id"`
	Stage         string    `json:"stage"`          // 执行阶段
	StageDate     *JSONTime `json:"stage_date"`     // 阶段日期
	Progress      float64   `json:"progress"`       // 进度百分比
	PaymentAmount float64   `json:"payment_amount"` // 付款金额
	PaymentDate   *JSONTime `json:"payment_date"`   // 付款日期
	Description   string    `json:"description"`    // 描述
}

// CreateContractExecution 创建执行记录
// 参数：input-创建输入，operatorID-操作人ID
// 返回：创建的执行记录和错误信息
func (s *ContractService) CreateContractExecution(input ContractExecutionCreateInput, operatorID uint) (*models.ContractExecution, error) {
	execution := models.ContractExecution{
		ContractID:    input.ContractID,
		Stage:         input.Stage,
		Progress:      input.Progress,
		PaymentAmount: input.PaymentAmount,
		Description:   input.Description,
		OperatorID:    operatorID,
	}

	if input.StageDate != nil && !input.StageDate.Time.IsZero() {
		execution.StageDate = &input.StageDate.Time
	}
	if input.PaymentDate != nil && !input.PaymentDate.Time.IsZero() {
		execution.PaymentDate = &input.PaymentDate.Time
	}

	if err := models.DB.Create(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// DeleteExecution 删除执行记录
// 参数：id-执行记录ID
// 返回：错误信息
func (s *ContractService) DeleteExecution(id uint) error {
	return models.DB.Delete(&models.ContractExecution{}, id).Error
}

// GetDocumentByID 根据ID获取文档
func (s *ContractService) GetDocumentByID(id uint) (*models.Document, error) {
	var document models.Document
	if err := models.DB.First(&document, id).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

// GetDocuments 获取合同的文档列表
// 参数：contractID-合同ID
// 返回：文档列表和错误信息
func (s *ContractService) GetDocuments(contractID uint) ([]models.Document, error) {
	var documents []models.Document
	if err := models.DB.Where("contract_id = ?", contractID).Order("created_at DESC").Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

// DocumentCreateInput 文档创建输入结构
type DocumentCreateInput struct {
	ContractID uint   `json:"contract_id"`
	Name       string `json:"name" binding:"required"`      // 文档名称
	FilePath   string `json:"file_path" binding:"required"` // 文件路径
	FileSize   int    `json:"file_size" binding:"required"` // 文件大小
	FileType   string `json:"file_type"`                    // 文件类型
}

// CreateDocument 创建文档记录
// 参数：input-创建输入，uploaderID-上传人ID
// 返回：创建的文档和错误信息
func (s *ContractService) CreateDocument(input DocumentCreateInput, uploaderID uint) (*models.Document, error) {
	document := models.Document{
		ContractID: input.ContractID,
		Name:       input.Name,
		FilePath:   input.FilePath,
		FileSize:   input.FileSize,
		FileType:   input.FileType,
		Version:    "1.0",
		UploaderID: uploaderID,
	}

	if err := models.DB.Create(&document).Error; err != nil {
		return nil, err
	}
	return &document, nil
}

// DeleteDocument 删除文档
// 参数：id-文档ID
// 返回：错误信息
func (s *ContractService) DeleteDocument(id uint) error {
	result := models.DB.Delete(&models.Document{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// LifecycleEventInput 生命周期事件输入结构
type LifecycleEventInput struct {
	EventType   string  `json:"event_type"`  // 事件类型
	FromStatus  string  `json:"from_status"` // 原状态
	ToStatus    string  `json:"to_status"`   // 目标状态
	Amount      float64 `json:"amount"`      // 涉及金额
	Description string  `json:"description"` // 描述
}

// AddLifecycleEvent 添加生命周期事件
// 记录合同的重要操作历史
// 参数：contractID-合同ID，input-事件输入，operatorID-操作人ID
// 返回：创建的事件和错误信息
func (s *ContractService) AddLifecycleEvent(contractID uint, input LifecycleEventInput, operatorID uint) (*models.ContractLifecycleEvent, error) {
	event := models.ContractLifecycleEvent{
		ContractID:  contractID,
		EventType:   models.LifecycleEventType(input.EventType),
		FromStatus:  input.FromStatus,
		ToStatus:    input.ToStatus,
		Amount:      input.Amount,
		Description: input.Description,
		OperatorID:  operatorID,
	}

	if err := models.DB.Create(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// GetLifecycleEvents 获取合同的生命周期事件列表
// 参数：contractID-合同ID
// 返回：事件列表和错误信息
func (s *ContractService) GetLifecycleEvents(contractID uint) ([]models.ContractLifecycleEvent, error) {
	var events []models.ContractLifecycleEvent
	if err := models.DB.Where("contract_id = ?", contractID).Order("created_at ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// UpdateContractStatus 更新合同状态
// 参数：contractID-合同ID，newStatus-新状态，operatorID-操作人ID
// 返回：更新后的合同和错误信息
func (s *ContractService) UpdateContractStatus(contractID uint, newStatus string, operatorID uint) (*models.Contract, error) {
	var contract models.Contract
	if err := models.DB.First(&contract, contractID).Error; err != nil {
		return nil, err
	}

	oldStatus := string(contract.Status)
	contract.Status = models.ContractStatus(newStatus)

	if err := models.DB.Save(&contract).Error; err != nil {
		return nil, err
	}

	// 添加状态变更生命周期事件
	s.AddLifecycleEvent(contractID, LifecycleEventInput{
		EventType:   "status_changed",
		FromStatus:  oldStatus,
		ToStatus:    newStatus,
		Description: "合同状态变更",
	}, operatorID)

	return &contract, nil
}

// ArchiveContract 归档合同
// 将合同状态设置为归档
// 参数：contractID-合同ID，operatorID-操作人ID
// 返回：归档后的合同和错误信息
func (s *ContractService) ArchiveContract(contractID uint, operatorID uint) (*models.Contract, error) {
	var contract models.Contract
	if err := models.DB.First(&contract, contractID).Error; err != nil {
		return nil, err
	}

	oldStatus := string(contract.Status)
	contract.Status = models.StatusArchived

	if err := models.DB.Save(&contract).Error; err != nil {
		return nil, err
	}

	s.AddLifecycleEvent(contractID, LifecycleEventInput{
		EventType:   string(models.LifecycleArchived),
		FromStatus:  oldStatus,
		ToStatus:    string(models.StatusArchived),
		Description: "合同已归档",
	}, operatorID)

	return &contract, nil
}

// StatusChangeRequireApproval 需要审批的状态列表
// 这些状态变更需要管理员审批才能生效
var StatusChangeRequireApproval = []string{
	"archived",    // 归档
	"terminated",  // 终止
	"in_progress", // 执行中
	"pending_pay", // 待付款
}

// IsStatusChangeRequireApproval 检查状态变更是否需要审批
// 参数：newStatus-目标状态
// 返回：是否需要审批
func (s *ContractService) IsStatusChangeRequireApproval(newStatus string) bool {
	for _, status := range StatusChangeRequireApproval {
		if status == newStatus {
			return true
		}
	}
	return false
}

// StatusChangeRequestInput 状态变更申请输入结构
type StatusChangeRequestInput struct {
	ToStatus string `json:"to_status" binding:"required"` // 目标状态
	Reason   string `json:"reason"`                       // 变更原因
}

// CreateStatusChangeRequest 创建状态变更申请
// 参数：contractID-合同ID，input-申请输入，requesterID-申请人ID
// 返回：创建的申请和错误信息，如果不需要审批则返回nil
func (s *ContractService) CreateStatusChangeRequest(contractID uint, input StatusChangeRequestInput, requesterID uint) (*models.StatusChangeRequest, error) {
	var contract models.Contract
	if err := models.DB.First(&contract, contractID).Error; err != nil {
		return nil, err
	}

	// 检查目标状态是否需要审批
	if !s.IsStatusChangeRequireApproval(input.ToStatus) {
		return nil, nil
	}

	// 检查是否已有待处理的申请
	var existingRequest models.StatusChangeRequest
	if err := models.DB.Where("contract_id = ? AND status = ?", contractID, "pending").First(&existingRequest).Error; err == nil {
		return nil, fmt.Errorf("该合同已有待审核的状态变更申请")
	}

	request := models.StatusChangeRequest{
		ContractID:  contractID,
		FromStatus:  string(contract.Status),
		ToStatus:    input.ToStatus,
		Reason:      input.Reason,
		RequesterID: requesterID,
		Status:      "pending",
	}

	if err := models.DB.Create(&request).Error; err != nil {
		return nil, err
	}

	return &request, nil
}

// GetStatusChangeRequests 获取合同的状态变更申请列表
// 参数：contractID-合同ID
// 返回：申请列表和错误信息
func (s *ContractService) GetStatusChangeRequests(contractID uint) ([]models.StatusChangeRequest, error) {
	var requests []models.StatusChangeRequest
	if err := models.DB.Preload("Requester").Preload("Approver").Where("contract_id = ?", contractID).Order("created_at DESC").Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// GetPendingStatusChangeRequests 获取待审批的状态变更申请
// 参数：role-用户角色
// 返回：申请列表和错误信息
func (s *ContractService) GetPendingStatusChangeRequests(role string) ([]models.StatusChangeRequest, error) {
	var requests []models.StatusChangeRequest
	query := models.DB.Preload("Contract.Customer").Preload("Requester").Order("created_at DESC")

	// 经理和管理员可以查看待审批的申请
	if role == "manager" || role == "admin" {
		query = query.Where("status = ?", "pending")
	}

	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// ApproveStatusChangeRequest 审批通过状态变更申请
// 参数：requestID-申请ID，approverID-审批人ID，comment-审批意见
// 返回：更新后的申请和错误信息
func (s *ContractService) ApproveStatusChangeRequest(requestID uint, approverID uint, comment string) (*models.StatusChangeRequest, error) {
	var request models.StatusChangeRequest
	if err := models.DB.Preload("Contract").First(&request, requestID).Error; err != nil {
		return nil, err
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("该申请已被处理")
	}

	now := time.Now()
	request.Status = "approved"
	request.ApproverID = &approverID
	request.Comment = comment
	request.ApprovedAt = &now

	if err := models.DB.Save(&request).Error; err != nil {
		return nil, err
	}

	// 更新合同状态
	var contract models.Contract
	if err := models.DB.First(&contract, request.ContractID).Error; err != nil {
		return nil, err
	}

	oldStatus := string(contract.Status)
	contract.Status = models.ContractStatus(request.ToStatus)
	if err := models.DB.Save(&contract).Error; err != nil {
		return nil, err
	}

	// 确定事件类型
	var eventType models.LifecycleEventType
	var description string
	switch request.ToStatus {
	case "archived":
		eventType = models.LifecycleArchived
		description = "合同已归档"
	case "terminated":
		eventType = models.LifecycleTerminated
		description = "合同已终止"
	default:
		eventType = "status_changed"
		description = "合同状态变更"
	}

	s.AddLifecycleEvent(request.ContractID, LifecycleEventInput{
		EventType:   string(eventType),
		FromStatus:  oldStatus,
		ToStatus:    request.ToStatus,
		Description: description,
	}, approverID)

	return &request, nil
}

// RejectStatusChangeRequest 拒绝状态变更申请
// 参数：requestID-申请ID，approverID-审批人ID，comment-拒绝原因
// 返回：更新后的申请和错误信息
func (s *ContractService) RejectStatusChangeRequest(requestID uint, approverID uint, comment string) (*models.StatusChangeRequest, error) {
	var request models.StatusChangeRequest
	if err := models.DB.First(&request, requestID).Error; err != nil {
		return nil, err
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("该申请已被处理")
	}

	now := time.Now()
	request.Status = "rejected"
	request.ApproverID = &approverID
	request.Comment = comment
	request.ApprovedAt = &now

	if err := models.DB.Save(&request).Error; err != nil {
		return nil, err
	}

	return &request, nil
}
