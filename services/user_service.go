package services

import (
	"contract-manage/models"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户服务结构体，提供用户相关的业务逻辑
type UserService struct{}

// NewUserService 创建用户服务实例
func NewUserService() *UserService {
	return &UserService{}
}

// TimePtr 将time.Time转换为*time.Time指针
// 用于解决GORM中时间指针的内存管理问题
func TimePtr(t time.Time) *time.Time {
	return &t
}

// CurrentTimePtr 返回当前时间的指针
func CurrentTimePtr() *time.Time {
	t := time.Now()
	return &t
}

// GetUserByID 根据用户ID获取用户信息
// 参数：id - 用户ID
// 返回：用户对象和错误信息
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := models.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
// 参数：username - 用户名
// 返回：用户对象和错误信息
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := models.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户信息
// 参数：email - 邮箱地址
// 返回：用户对象和错误信息
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := models.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUsers 分页获取用户列表
// 参数：skip - 跳过的记录数, limit - 返回的记录数限制
//
//	username - 可选的用户名搜索, role - 可选的角色筛选
//
// 返回：用户列表和错误信息
func (s *UserService) GetUsers(skip, limit int, username, role string) ([]models.User, error) {
	var users []models.User
	query := models.DB
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if err := query.Offset(skip).Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}

	// 查询每个用户的锁定状态
	for i := range users {
		users[i].IsLocked = s.IsAccountLocked(users[i].Username)
	}

	return users, nil
}

// GetUsersCount 获取用户总数
func (s *UserService) GetUsersCount(username, role string) (int64, error) {
	var count int64
	query := models.DB.Model(&models.User{})
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UserCreateInput 创建用户的输入结构体
type UserCreateInput struct {
	Username          string                `json:"username" binding:"required"`
	Email             string                `json:"email"`
	Password          string                `json:"password" binding:"required"`
	FullName          string                `json:"full_name"`
	Role              models.UserRole       `json:"role"`
	CustomPermissions string                `json:"custom_permissions"` // 用户自定义权限（JSON数组格式）
	Department        string                `json:"department"`
	Phone             string                `json:"phone"`
	AccountStatus     models.UserStatusType `json:"account_status"` // 账号状态类型
	ValidFrom         *string               `json:"valid_from"`     // 有效期开始时间
	ValidTo           *string               `json:"valid_to"`       // 有效期结束时间
	ValidHours        int                   `json:"valid_hours"`    // 临时账号有效期小时数
}

// CreateUser 创建新用户
// 功能说明：
//   - 检查用户名和邮箱是否已被注册
//   - 使用bcrypt对密码进行加密
//   - 计算用户完整性哈希值用于数据完整性验证
//   - 创建用户记录并返回
func (s *UserService) CreateUser(input UserCreateInput) (*models.User, error) {
	// 检查用户名是否已存在
	if _, err := s.GetUserByUsername(input.Username); err == nil {
		return nil, errors.New("username already registered")
	}
	// 检查邮箱是否已存在（如果提供了邮箱）
	if input.Email != "" {
		if _, err := s.GetUserByEmail(input.Email); err == nil {
			return nil, errors.New("email already registered")
		}
	}

	// 使用bcrypt对密码进行哈希加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 计算密码的SHA-256杂凑值，用于前端传输验证
	passwordHash := models.CalculatePasswordHash(input.Password)

	// 设置默认账号状态
	accountStatus := models.UserStatusPermanent
	if input.AccountStatus != "" {
		accountStatus = input.AccountStatus
	}

	// 解析有效期时间
	validFrom := (*time.Time)(nil)
	validTo := (*time.Time)(nil)

	if input.ValidFrom != nil && *input.ValidFrom != "" {
		if t, err := time.Parse("2006-01-02", *input.ValidFrom); err == nil {
			validFrom = TimePtr(t)
		}
	}
	if input.ValidTo != nil && *input.ValidTo != "" {
		if t, err := time.Parse("2006-01-02", *input.ValidTo); err == nil {
			validTo = TimePtr(t)
		}
	}

	// 临时账号默认有效期小时数
	validHours := input.ValidHours
	if accountStatus == models.UserStatusTemporary {
		if validHours <= 0 {
			validHours = 24 // 默认24小时
		}
		// 临时账号自动设置开始时间为当前时间
		if validFrom == nil {
			validFrom = CurrentTimePtr()
		}
	}

	// 创建用户对象，包含完整性哈希值和密码SHA-256杂凑值
	// 完整性哈希值由 用户名:邮箱:哈希密码 组成，用于验证用户数据的完整性
	user := models.User{
		Username:          input.Username,
		Email:             input.Email,
		HashedPassword:    string(hashedPassword),
		PasswordHash:      passwordHash,
		HashVerified:      true,
		IntegrityHash:     models.CalculateUserIntegrityHash(input.Username, input.Email, string(hashedPassword)),
		FullName:          input.FullName,
		Role:              input.Role,
		CustomPermissions: input.CustomPermissions,
		Department:        input.Department,
		Phone:             input.Phone,
		IsActive:          true,
		AccountStatus:     accountStatus,
		ValidFrom:         validFrom,
		ValidTo:           validTo,
		ValidHours:        validHours,
	}

	// 保存用户到数据库
	if err := models.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UserUpdateInput 更新用户信息的输入结构体
type UserUpdateInput struct {
	Email             string                `json:"email"`
	FullName          string                `json:"full_name"`
	Role              models.UserRole       `json:"role"`
	CustomPermissions string                `json:"custom_permissions"` // 用户自定义权限（JSON数组格式）
	Department        string                `json:"department"`
	Phone             string                `json:"phone"`
	IsActive          *bool                 `json:"is_active"`
	Password          string                `json:"password"`
	AccountStatus     models.UserStatusType `json:"account_status"` // 账号状态类型
	ValidFrom         *string               `json:"valid_from"`     // 有效期开始时间
	ValidTo           *string               `json:"valid_to"`       // 有效期结束时间
	ValidHours        int                   `json:"valid_hours"`    // 临时账号有效期小时数
}

// UpdateUser 更新用户信息
// 功能说明：
//   - 根据ID获取用户信息
//   - 更新用户提供的字段
//   - 如果更新了密码，重新计算完整性哈希值
func (s *UserService) UpdateUser(id uint, input UserUpdateInput) (*models.User, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	// 构建更新字段映射
	updates := map[string]interface{}{}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if input.FullName != "" {
		updates["full_name"] = input.FullName
	}
	if input.Role != "" {
		updates["role"] = input.Role
	}
	// 更新自定义权限
	if input.CustomPermissions != "" || input.CustomPermissions == "[]" {
		updates["custom_permissions"] = input.CustomPermissions
	}
	if input.Department != "" {
		updates["department"] = input.Department
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	// 更新账号状态
	if input.AccountStatus != "" {
		updates["account_status"] = input.AccountStatus

		// 如果设置为临时账号，自动设置开始时间为当前时间
		if models.UserStatusType(input.AccountStatus) == models.UserStatusTemporary {
			if input.ValidHours > 0 {
				updates["valid_hours"] = input.ValidHours
			} else {
				updates["valid_hours"] = 24 // 默认24小时
			}
			now := time.Now()
			updates["valid_from"] = now
		}
	}

	// 解析并更新有效期时间
	if input.ValidFrom != nil {
		if *input.ValidFrom == "" {
			updates["valid_from"] = nil
		} else if t, err := time.Parse("2006-01-02", *input.ValidFrom); err == nil {
			updates["valid_from"] = t
		}
	}
	if input.ValidTo != nil {
		if *input.ValidTo == "" {
			updates["valid_to"] = nil
		} else if t, err := time.Parse("2006-01-02", *input.ValidTo); err == nil {
			updates["valid_to"] = t
		}
	}

	// 更新临时账号有效期小时数（如果没有在账号状态更新时设置）
	if input.ValidHours > 0 && input.AccountStatus == "" {
		updates["valid_hours"] = input.ValidHours
	}

	// 如果提供了新密码，更新密码和完整性哈希值
	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.HashedPassword = string(hashedPassword)
		updates["hashed_password"] = string(hashedPassword)
		// 计算新密码的SHA-256杂凑值
		passwordHash := models.CalculatePasswordHash(input.Password)
		updates["password_hash"] = passwordHash
		updates["hash_verified"] = true
		// 重新计算完整性哈希值，包含新的密码哈希
		updates["integrity_hash"] = models.CalculateUserIntegrityHash(user.Username, user.Email, string(hashedPassword))
	}

	// 执行更新操作
	if err := models.DB.Model(user).Updates(updates).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser 删除用户
// 参数：id - 用户ID
// 返回：错误信息，如果记录不存在返回gorm.ErrRecordNotFound
func (s *UserService) DeleteUser(id uint) error {
	result := models.DB.Delete(&models.User{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// AuthenticateUser 用户认证
// 功能说明：
//   - 根据用户名获取用户信息
//   - 使用bcrypt验证密码
//   - 验证用户数据完整性（通过IntegrityHash）
//
// 参数：username - 用户名, password - 明文密码
// 返回：认证成功返回用户对象，失败返回错误信息
func (s *UserService) AuthenticateUser(username, password string) (*models.User, error) {
	// 根据用户名获取用户
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	// 使用bcrypt比较密码哈希值
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		return nil, errors.New("invalid password")
	}
	// 如果IntegrityHash为空，自动计算并更新
	if user.IntegrityHash == "" {
		user.IntegrityHash = models.CalculateUserIntegrityHash(user.Username, user.Email, user.HashedPassword)
		models.DB.Model(user).Updates(map[string]interface{}{"integrity_hash": user.IntegrityHash})
	}
	// 验证用户数据的完整性
	// 通过重新计算哈希值并与存储的IntegrityHash比较
	if !user.VerifyIntegrity() {
		return nil, errors.New("user data integrity check failed")
	}
	return user, nil
}

// IsAccountLocked 检查账号是否被锁定
func (s *UserService) IsAccountLocked(username string) bool {
	var record models.LoginFailureRecord
	username = strings.TrimSpace(username)
	if err := models.DB.Where("TRIM(username) = ?", username).First(&record).Error; err != nil {
		return false
	}

	// 检查是否被手动锁定
	if record.Locked {
		// 如果设定了锁定截止时间
		if record.LockedUntil != nil {
			if time.Now().After(*record.LockedUntil) {
				// 锁定已过期，解除锁定
				s.ClearLoginFailures(username)
				return false
			}
			return true
		}
		return true
	}

	// 检查失败次数是否超过阈值
	if record.FailCount >= models.MaxLoginAttempts {
		// 自动锁定
		lockedUntil := time.Now().Add(models.LockoutDuration)
		models.DB.Model(&record).Updates(map[string]interface{}{
			"locked":       true,
			"locked_until": lockedUntil,
		})
		return true
	}

	return false
}

// RecordLoginFailure 记录登录失败
func (s *UserService) RecordLoginFailure(username string) {
	var record models.LoginFailureRecord
	now := time.Now()

	if err := models.DB.Where("username = ?", username).First(&record).Error; err != nil {
		// 记录不存在，创建新记录
		record = models.LoginFailureRecord{
			Username:  username,
			FailCount: 1,
			FirstFail: now,
			LastFail:  now,
			Locked:    false,
		}
		models.DB.Create(&record)
	} else {
		// 检查是否需要重置失败计数（超过30分钟）
		if now.Sub(record.LastFail) > models.FailureResetMinutes*time.Minute {
			record.FailCount = 1
			record.FirstFail = now
		} else {
			record.FailCount++
		}
		record.LastFail = now

		// 如果超过阈值，自动锁定
		if record.FailCount >= models.MaxLoginAttempts {
			lockedUntil := now.Add(models.LockoutDuration)
			models.DB.Model(&record).Updates(map[string]interface{}{
				"fail_count":   record.FailCount,
				"last_fail":    record.LastFail,
				"locked":       true,
				"locked_until": lockedUntil,
			})
		} else {
			models.DB.Model(&record).Updates(map[string]interface{}{
				"fail_count": record.FailCount,
				"last_fail":  record.LastFail,
			})
		}
	}
}

// ClearLoginFailures 清除登录失败记录
func (s *UserService) ClearLoginFailures(username string) {
	username = strings.TrimSpace(username)
	models.DB.Where("TRIM(username) = ?", username).Delete(&models.LoginFailureRecord{})
}

// GetLoginFailureInfo 获取登录失败信息
func (s *UserService) GetLoginFailureInfo(username string) (int, bool) {
	var record models.LoginFailureRecord
	username = strings.TrimSpace(username)
	if err := models.DB.Where("TRIM(username) = ?", username).First(&record).Error; err != nil {
		return 0, false
	}
	return record.FailCount, record.Locked
}

// ResetUserPassword 重置用户密码
// 参数：userID - 用户ID, newPassword - 新密码
// 返回：错误信息
func (s *UserService) ResetUserPassword(userID uint, newPassword string) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// 使用bcrypt加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 计算新密码的SHA-256杂凑值
	passwordHash := models.CalculatePasswordHash(newPassword)

	// 重新计算完整性哈希值
	integrityHash := models.CalculateUserIntegrityHash(user.Username, user.Email, string(hashedPassword))

	// 更新用户密码相关信息
	updates := map[string]interface{}{
		"hashed_password": string(hashedPassword),
		"password_hash":   passwordHash,
		"hash_verified":   true,
		"integrity_hash":  integrityHash,
	}

	if err := models.DB.Model(user).Updates(updates).Error; err != nil {
		return err
	}

	// 清除该用户的登录失败记录
	s.ClearLoginFailures(user.Username)

	return nil
}
