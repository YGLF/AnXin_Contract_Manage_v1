package services

import (
	"contract-manage/models"
	"time"
)

// AuditService 审计日志服务结构体，提供操作审计相关的业务逻辑
type AuditService struct{}

// NewAuditService 创建审计日志服务实例
func NewAuditService() *AuditService {
	return &AuditService{}
}

// CreateAuditLog 创建审计日志记录
// 功能说明：
//   - 将用户操作记录到审计日志
//   - 包括操作人、操作类型、模块、IP地址等信息
//
// 参数：log - 审计日志对象
func (s *AuditService) CreateAuditLog(log models.AuditLog) error {
	return models.DB.Create(&log).Error
}

// GetAuditLogs 分页查询审计日志
// 功能说明：
//   - 支持按用户名、操作类型、模块进行模糊搜索
//   - 支持按日期范围过滤
//   - 返回符合条件的日志列表和总数
//
// 参数说明：
//   - page: 页码（从1开始）
//   - pageSize: 每页记录数
//   - username: 用户名搜索关键词
//   - action: 操作类型搜索关键词
//   - module: 模块名称精确匹配
//   - startDate: 开始日期（格式：2006-01-02）
//   - endDate: 结束日期（格式：2006-01-02）
//
// 返回：日志列表、总记录数和错误信息
func (s *AuditService) GetAuditLogs(page, pageSize int, username, action, module, startDate, endDate string, statusCode *int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := models.DB.Model(&models.AuditLog{})

	// 添加用户名模糊搜索条件
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	// 添加操作类型模糊搜索条件
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}
	// 添加模块精确匹配条件
	if module != "" {
		query = query.Where("module = ?", module)
	}
	// 添加状态码筛选条件
	if statusCode != nil && *statusCode > 0 {
		if *statusCode >= 400 {
			query = query.Where("status_code >= ?", *statusCode)
		} else {
			query = query.Where("status_code = ?", *statusCode)
		}
	}
	// 添加开始日期条件
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	// 添加结束日期条件（包含当天）
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(24 * time.Hour)
			query = query.Where("created_at < ?", t)
		}
	}

	// 统计符合条件的记录总数
	query.Count(&total)

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	// 按创建时间倒序分页查询
	err := query.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// DeleteAuditLog 删除指定ID的审计日志
// 参数：id - 审计日志ID
// 返回：错误信息
func (s *AuditService) DeleteAuditLog(id uint) error {
	return models.DB.Delete(&models.AuditLog{}, id).Error
}

// DeleteAuditLogs 批量删除审计日志
// 功能说明：
//   - 根据提供的ID列表批量删除审计日志
//   - 如果列表为空，直接返回成功
//
// 参数：ids - 要删除的审计日志ID列表
// 返回：错误信息
func (s *AuditService) DeleteAuditLogs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return models.DB.Where("id IN ?", ids).Delete(&models.AuditLog{}).Error
}

// GetAuditStatistics 获取审计统计数据
// 功能说明：
//   - 统计指定日期范围内的日志总数
//   - 统计登录操作的次数
//   - 统计各操作类型的次数分布（前10名）
//
// 参数说明：
//   - startDate: 开始日期（格式：2006-01-02）
//   - endDate: 结束日期（格式：2006-01-02）
//
// 返回：统计数据字典，包含total、loginCount和actionCounts
func (s *AuditService) GetAuditStatistics(startDate, endDate string) (map[string]interface{}, error) {
	var total int64
	var loginCount int64
	var actionCounts []struct {
		Action string
		Count  int64
	}

	query := models.DB.Model(&models.AuditLog{})

	// 添加开始日期条件
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	// 添加结束日期条件
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(24 * time.Hour)
			query = query.Where("created_at < ?", t)
		}
	}

	// 统计日志总数
	query.Count(&total)

	// 统计登录操作次数（包含login关键词的操作）
	models.DB.Model(&models.AuditLog{}).Where("action LIKE ?", "%login%").Count(&loginCount)

	// 统计各操作类型的次数，按次数倒序，取前10名
	models.DB.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Order("count DESC").
		Limit(10).
		Scan(&actionCounts)

	stats := map[string]interface{}{
		"total":        total,
		"loginCount":   loginCount,
		"actionCounts": actionCounts,
	}

	return stats, nil
}

// RecordLogin 记录登录操作
// 功能说明：
//   - 根据登录成功/失败状态设置不同的操作描述和状态码
//   - 创建审计日志记录登录行为
//
// 参数说明：
//   - userID: 用户ID（登录失败时为0）
//   - username: 用户名
//   - ipAddress: 客户端IP地址
//   - userAgent: 客户端User-Agent信息
//   - success: 登录是否成功
func (s *AuditService) RecordLogin(userID uint, username, ipAddress, userAgent string, success bool) {
	action := "POST /api/auth/login"
	if !success {
		action = "POST /api/auth/login (failed)"
	}

	// 创建审计日志对象
	log := models.AuditLog{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Module:    "auth",
		Method:    "POST",
		Path:      "/api/auth/login",
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// 根据登录结果设置HTTP状态码
	if success {
		log.StatusCode = 200
	} else {
		log.StatusCode = 401
	}

	// 保存到数据库
	models.DB.Create(&log)
}

// RecordLoginSuccess 记录成功登录
// 功能说明：
//   - 便捷函数，用于记录登录成功事件
//   - 内部调用RecordLogin方法
//
// 参数：userID - 用户ID, username - 用户名, ipAddress - IP地址, userAgent - User-Agent
func RecordLoginSuccess(userID uint, username, ipAddress, userAgent string) {
	s := NewAuditService()
	s.RecordLogin(userID, username, ipAddress, userAgent, true)
}

// RecordLoginFailure 记录失败登录
// 功能说明：
//   - 便捷函数，用于记录登录失败事件
//   - 内部调用RecordLogin方法
//   - userID设为0
//
// 参数：username - 用户名, ipAddress - IP地址, userAgent - User-Agent
func RecordLoginFailure(username, ipAddress, userAgent string) {
	s := NewAuditService()
	s.RecordLogin(0, username, ipAddress, userAgent, false)
}
