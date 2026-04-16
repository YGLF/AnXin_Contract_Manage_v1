package handlers

import (
	"contract-manage/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CustomerHandler 客户/供应商处理器
// 处理客户信息和合同类型的CRUD操作
type CustomerHandler struct {
	customerService *services.CustomerService // 客户服务实例
}

// NewCustomerHandler 创建客户处理器实例
// 返回：配置好的CustomerHandler指针
func NewCustomerHandler() *CustomerHandler {
	return &CustomerHandler{
		customerService: services.NewCustomerService(),
	}
}

// GetCustomers 获取客户列表处理器
// 返回分页的客户/供应商列表，支持按类型筛选
// GET /api/customers
func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	// 解析分页参数
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))     // 跳过记录数，默认0
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100")) // 每页数量，默认100
	customerType := c.Query("type")                          // 可选：筛选客户类型
	name := c.Query("name")                                  // 可选：按名称搜索

	// 调用服务层获取客户列表
	customers, err := h.customerService.GetCustomers(skip, limit, customerType, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取总数
	count, _ := h.customerService.GetCustomersCount(customerType, name)

	c.JSON(http.StatusOK, gin.H{
		"data":  customers,
		"total": count,
	})
}

// GetCustomerByID 根据ID获取客户详情处理器
// 返回指定客户的详细信息
// GET /api/customers/:customer_id
func (h *CustomerHandler) GetCustomerByID(c *gin.Context) {
	// 解析客户ID参数
	id, err := strconv.ParseUint(c.Param("customer_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// 调用服务层获取客户信息
	customer, err := h.customerService.GetCustomerByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// CreateCustomer 创建客户处理器
// 添加新的客户或供应商信息
// POST /api/customers
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	// 解析请求体
	var input services.CustomerCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 默认类型为customer（客户）
	if input.Type == "" {
		input.Type = "customer"
	}

	// 调用服务层创建客户
	customer, err := h.customerService.CreateCustomer(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

// UpdateCustomer 更新客户信息处理器
// 修改客户的基本信息
// PUT /api/customers/:customer_id
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	// 解析客户ID参数
	id, err := strconv.ParseUint(c.Param("customer_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// 解析请求体
	var input services.CustomerUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务层更新客户信息
	customer, err := h.customerService.UpdateCustomer(uint(id), input)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// DeleteCustomer 删除客户处理器
// 删除指定的客户记录
// DELETE /api/customers/:customer_id
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	// 解析客户ID参数
	id, err := strconv.ParseUint(c.Param("customer_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	// 调用服务层删除客户
	if err := h.customerService.DeleteCustomer(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetContractTypes 获取合同类型列表处理器
// 返回所有合同类型，支持分页
// GET /api/contract-types
func (h *CustomerHandler) GetContractTypes(c *gin.Context) {
	// 解析分页参数
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	// 调用服务层获取合同类型列表
	contractTypes, err := h.customerService.GetContractTypes(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contractTypes)
}

// CreateContractType 创建合同类型处理器
// 添加新的合同类型
// POST /api/contract-types
func (h *CustomerHandler) CreateContractType(c *gin.Context) {
	// 解析请求体
	var input services.ContractTypeCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务层创建合同类型
	contractType, err := h.customerService.CreateContractType(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, contractType)
}

// UpdateContractType 更新合同类型处理器
// 修改合同类型的名称和描述
// PUT /api/contract-types/:type_id
func (h *CustomerHandler) UpdateContractType(c *gin.Context) {
	// 解析类型ID参数
	id, err := strconv.ParseUint(c.Param("type_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 解析请求体
	var input services.ContractTypeCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务层更新合同类型
	contractType, err := h.customerService.UpdateContractType(uint(id), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contractType)
}

// DeleteContractType 删除合同类型处理器
// 删除指定的合同类型
// DELETE /api/contract-types/:type_id
func (h *CustomerHandler) DeleteContractType(c *gin.Context) {
	// 解析类型ID参数
	id, err := strconv.ParseUint(c.Param("type_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 调用服务层删除合同类型
	if err := h.customerService.DeleteContractType(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
