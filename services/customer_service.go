package services

import (
	"contract-manage/models"
	"errors"

	"gorm.io/gorm"
)

// CustomerService 客户管理服务结构体，提供客户相关的业务逻辑
type CustomerService struct{}

// NewCustomerService 创建客户管理服务实例
func NewCustomerService() *CustomerService {
	return &CustomerService{}
}

// GetCustomerByID 根据客户ID获取客户信息
// 参数：id - 客户ID
// 返回：客户对象和错误信息
func (s *CustomerService) GetCustomerByID(id uint) (*models.Customer, error) {
	var customer models.Customer
	if err := models.DB.First(&customer, id).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetCustomerByCode 根据客户编码获取客户信息
// 参数：code - 客户编码
// 返回：客户对象和错误信息
func (s *CustomerService) GetCustomerByCode(code string) (*models.Customer, error) {
	var customer models.Customer
	if err := models.DB.Where("code = ?", code).First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetCustomers 分页获取客户列表
// 参数说明：
//   - skip: 跳过的记录数，用于分页
//   - limit: 返回的记录数限制
//   - customerType: 可选的客户类型过滤条件
//   - name: 可选的名称搜索条件
//
// 返回：符合条件的客户列表和错误信息
func (s *CustomerService) GetCustomers(skip, limit int, customerType, name string) ([]models.Customer, error) {
	var customers []models.Customer
	query := models.DB
	// 如果提供了客户类型，添加过滤条件
	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}
	// 如果提供了名称，添加模糊搜索
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if err := query.Offset(skip).Limit(limit).Find(&customers).Error; err != nil {
		return nil, err
	}
	return customers, nil
}

// GetCustomersCount 获取客户总数
func (s *CustomerService) GetCustomersCount(customerType, name string) (int64, error) {
	var count int64
	query := models.DB.Model(&models.Customer{})
	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CustomerCreateInput 创建客户的输入结构体
type CustomerCreateInput struct {
	Name          string `json:"name" binding:"required"`
	Type          string `json:"type"`
	Code          string `json:"code" binding:"required"`
	ContactPerson string `json:"contact_person"`
	ContactPhone  string `json:"contact_phone"`
	ContactEmail  string `json:"contact_email"`
	Address       string `json:"address"`
	CreditRating  string `json:"credit_rating"`
}

// CreateCustomer 创建新客户
// 功能说明：
//   - 检查客户编码是否已存在
//   - 创建客户记录并返回
func (s *CustomerService) CreateCustomer(input CustomerCreateInput) (*models.Customer, error) {
	// 检查客户编码是否已存在
	if _, err := s.GetCustomerByCode(input.Code); err == nil {
		return nil, errors.New("customer code already exists")
	}

	// 创建客户对象
	customer := models.Customer{
		Name:          input.Name,
		Type:          input.Type,
		Code:          input.Code,
		ContactPerson: input.ContactPerson,
		ContactPhone:  input.ContactPhone,
		ContactEmail:  input.ContactEmail,
		Address:       input.Address,
		CreditRating:  input.CreditRating,
		IsActive:      true,
	}

	// 保存到数据库
	if err := models.DB.Create(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

// CustomerUpdateInput 更新客户信息的输入结构体
type CustomerUpdateInput struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	ContactPerson string `json:"contact_person"`
	ContactPhone  string `json:"contact_phone"`
	ContactEmail  string `json:"contact_email"`
	Address       string `json:"address"`
	CreditRating  string `json:"credit_rating"`
	IsActive      *bool  `json:"is_active"`
}

// UpdateCustomer 更新客户信息
// 功能说明：
//   - 根据ID获取客户信息
//   - 更新客户提供的字段
//   - 只更新非空字段，保留原值
func (s *CustomerService) UpdateCustomer(id uint, input CustomerUpdateInput) (*models.Customer, error) {
	customer, err := s.GetCustomerByID(id)
	if err != nil {
		return nil, err
	}

	// 构建更新字段映射
	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Type != "" {
		updates["type"] = input.Type
	}
	if input.ContactPerson != "" {
		updates["contact_person"] = input.ContactPerson
	}
	if input.ContactPhone != "" {
		updates["contact_phone"] = input.ContactPhone
	}
	if input.ContactEmail != "" {
		updates["contact_email"] = input.ContactEmail
	}
	if input.Address != "" {
		updates["address"] = input.Address
	}
	if input.CreditRating != "" {
		updates["credit_rating"] = input.CreditRating
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	// 执行更新操作
	if err := models.DB.Model(customer).Updates(updates).Error; err != nil {
		return nil, err
	}
	return customer, nil
}

// DeleteCustomer 删除客户
// 参数：id - 客户ID
// 返回：错误信息，如果记录不存在返回gorm.ErrRecordNotFound
func (s *CustomerService) DeleteCustomer(id uint) error {
	result := models.DB.Delete(&models.Customer{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// GetContractTypeByID 根据合同类型ID获取合同类型信息
// 参数：id - 合同类型ID
// 返回：合同类型对象和错误信息
func (s *CustomerService) GetContractTypeByID(id uint) (*models.ContractType, error) {
	var contractType models.ContractType
	if err := models.DB.First(&contractType, id).Error; err != nil {
		return nil, err
	}
	return &contractType, nil
}

// GetContractTypes 分页获取合同类型列表
// 参数说明：
//   - skip: 跳过的记录数
//   - limit: 返回的记录数限制
//
// 返回：合同类型列表和错误信息
func (s *CustomerService) GetContractTypes(skip, limit int) ([]models.ContractType, error) {
	var contractTypes []models.ContractType
	if err := models.DB.Offset(skip).Limit(limit).Find(&contractTypes).Error; err != nil {
		return nil, err
	}
	return contractTypes, nil
}

// ContractTypeCreateInput 创建合同类型的输入结构体
type ContractTypeCreateInput struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

// CreateContractType 创建新的合同类型
// 参数：input - 包含合同类型名称、编码和描述的输入结构
// 返回：创建的合同类型对象和错误信息
func (s *CustomerService) CreateContractType(input ContractTypeCreateInput) (*models.ContractType, error) {
	contractType := models.ContractType{
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
	}

	if err := models.DB.Create(&contractType).Error; err != nil {
		return nil, err
	}
	return &contractType, nil
}

// UpdateContractType 更新合同类型信息
// 参数说明：
//   - id: 合同类型ID
//   - input: 包含更新后名称、编码和描述的输入结构
//
// 返回：更新后的合同类型对象和错误信息
func (s *CustomerService) UpdateContractType(id uint, input ContractTypeCreateInput) (*models.ContractType, error) {
	var contractType models.ContractType
	if err := models.DB.First(&contractType, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":        input.Name,
		"code":        input.Code,
		"description": input.Description,
	}

	if err := models.DB.Model(&contractType).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &contractType, nil
}

// DeleteContractType 删除合同类型
// 参数：id - 合同类型ID
// 返回：错误信息
func (s *CustomerService) DeleteContractType(id uint) error {
	return models.DB.Delete(&models.ContractType{}, id).Error
}
