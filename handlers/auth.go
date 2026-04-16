package handlers

import (
	"contract-manage/config"
	"contract-manage/middleware"
	"contract-manage/models"
	"contract-manage/services"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
// 处理用户注册、登录、用户管理等认证相关请求
type AuthHandler struct {
	userService *services.UserService // 用户服务，用于操作用户数据
}

// NewAuthHandler 创建认证处理器实例
// 返回：配置好的AuthHandler指针
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userService: services.NewUserService(),
	}
}

// LoginRequest 登录请求结构
// 客户端提交的登录信息
type LoginRequest struct {
	Username     string `json:"username" binding:"required"` // 用户名，必填
	Password     string `json:"password" binding:"required"` // 密码（明文，用于后端bcrypt验证）
	PasswordHash string `json:"password_hash"`               // 密码SHA-256杂凑值（前端发送，用于杂凑比对）
}

// TokenResponse 登录响应结构
// 登录成功返回的令牌和用户信息
type TokenResponse struct {
	AccessToken   string                `json:"access_token"`  // JWT访问令牌
	ExpiresIn     int                   `json:"expires_in"`    // 令牌过期时间（秒）
	TokenType     string                `json:"token_type"`    // 令牌类型，通常为"bearer"
	UserInfo      *UserInfo             `json:"user_info"`     // 用户基本信息
	UnreadCount   int64                 `json:"unread_count"`  // 未读通知数量
	Notifications []models.Notification `json:"notifications"` // 通知列表
}

// UserInfo 用户基本信息结构
// 用于返回给客户端的用户信息（不含敏感数据）
type UserInfo struct {
	ID          uint     `json:"id"`          // 用户ID
	Username    string   `json:"username"`    // 用户名
	Email       string   `json:"email"`       // 邮箱
	FullName    string   `json:"full_name"`   // 真实姓名
	Role        string   `json:"role"`        // 用户角色
	Permissions []string `json:"permissions"` // 用户权限列表
}

// 邮箱格式正则表达式
// 标准邮箱格式：用户名@域名.顶级域名
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\x{4e00}-\x{9fa5}]{3,20}$`)
)

// sanitizeInput 输入消毒函数
// 移除可能导致XSS或注入攻击的危险字符
// 参数：input-原始输入字符串
// 返回：消毒后的安全字符串
func sanitizeInput(input string) string {
	// HTML转义，防止XSS攻击
	// 使用更精确的转义规则，不影响正常的中文字符
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	return input
}

// validateUsername 验证用户名格式
// 检查用户名长度和字符合法性
// 参数：username-待验证的用户名
// 返回：验证失败返回错误信息
func validateUsername(username string) error {
	// 检查是否为空
	if username == "" {
		return usernameError("用户名不能为空")
	}
	// 检查长度：3-20个字符
	if len(username) < 3 || len(username) > 20 {
		return usernameError("用户名长度必须在3-20个字符之间")
	}
	// 检查字符：只允许字母、数字、下划线
	if !usernameRegex.MatchString(username) {
		return usernameError("用户名只能包含字母、数字、下划线和中文")
	}
	return nil
}

// validateEmail 验证邮箱格式
// 参数：email-待验证的邮箱地址
// 返回：验证失败返回错误信息
func validateEmail(email string) error {
	// 邮箱可以为空（可选字段）
	if email == "" {
		return nil
	}
	// 格式检查
	if !emailRegex.MatchString(email) {
		return usernameError("邮箱格式不正确")
	}
	return nil
}

// validatePassword 验证密码强度
// 参数：password-待验证的密码
// 返回：验证失败返回错误信息
func validatePassword(password string) error {
	// 检查是否为空
	if password == "" {
		return usernameError("密码不能为空")
	}
	// 检查最小长度：至少6位
	if len(password) < 6 {
		return usernameError("密码长度至少6位")
	}
	// 检查最大长度
	if len(password) > 50 {
		return usernameError("密码长度不能超过50位")
	}
	return nil
}

// validationError 自定义验证错误类型
// 用于返回格式化的错误信息
type validationError struct {
	message string // 错误消息
}

// Error 实现error接口，返回错误消息
func (e validationError) Error() string {
	return e.message
}

// usernameError 创建验证错误
// 参数：msg-错误消息
// 返回：validationError实例
func usernameError(msg string) error {
	return validationError{message: msg}
}

// Register 用户注册处理器
// 处理新用户注册请求
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	// 解析请求体
	var input services.UserCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式不正确"})
		return
	}

	// 对所有输入进行消毒处理，防止XSS攻击
	input.Username = sanitizeInput(input.Username)
	input.Email = sanitizeInput(input.Email)
	input.FullName = sanitizeInput(input.FullName)
	input.Department = sanitizeInput(input.Department)
	input.Phone = sanitizeInput(input.Phone)

	// 验证用户名格式
	if err := validateUsername(input.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证邮箱格式
	if err := validateEmail(input.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证密码强度
	if err := validatePassword(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果未指定角色，默认设置为普通用户
	if input.Role == "" {
		input.Role = "user"
	}

	// 调用服务层创建用户
	user, err := h.userService.CreateUser(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"message":  "注册成功",
	})
}

// Login 用户登录处理器
// 处理用户登录请求，验证身份并返回JWT令牌
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	// 解析登录请求
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式不正确"})
		return
	}

	// 验证用户名不能为空
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	// 消毒用户名输入
	req.Username = sanitizeInput(req.Username)

	// 登录时不需要验证用户名格式（用户名已在注册时验证）

	// 检查登录失败锁定
	if h.userService.IsAccountLocked(req.Username) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":  "账号已被锁定，请稍后再试",
			"code":   "ACCOUNT_LOCKED",
			"locked": true,
		})
		return
	}

	// 调用服务层验证用户身份（使用bcrypt验证）
	user, err := h.userService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		// 记录登录失败
		h.userService.RecordLoginFailure(req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
			"code":  "INVALID_CREDENTIALS",
		})
		return
	}

	// 检查用户账号是否被禁用
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "账号已被禁用",
			"code":  "ACCOUNT_DISABLED",
		})
		return
	}

	// 检查账号有效期状态
	if !user.IsAccountValid() {
		var errorMsg string
		switch user.AccountStatus {
		case models.UserStatusDisabled:
			errorMsg = "账号已被禁用"
		case models.UserStatusTemporary:
			if user.ValidHours > 0 {
				errorMsg = "临时账号已过期"
			} else {
				errorMsg = "临时账号已过期"
			}
		case models.UserStatusTimed:
			errorMsg = "账号不在有效期内"
		default:
			errorMsg = "账号已过期或不在有效期内"
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error": errorMsg,
			"code":  "ACCOUNT_EXPIRED",
		})
		return
	}

	// 登录成功后清除失败记录
	h.userService.ClearLoginFailures(req.Username)

	// 验证密码SHA-256杂凑值
	if req.PasswordHash != "" {
		// 如果前端发送了SHA-256杂凑值，与数据库存储的杂凑值进行比对
		if user.PasswordHash != "" && req.PasswordHash == user.PasswordHash {
			// 更新杂凑验证状态为一致
			if !user.HashVerified {
				models.DB.Model(user).Updates(map[string]interface{}{"hash_verified": true})
			}
		} else {
			// 杂凑不一致，更新状态并返回错误
			models.DB.Model(user).Updates(map[string]interface{}{"hash_verified": false})
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "密码验证不一致",
				"code":  "HASH_MISMATCH",
			})
			return
		}
	}

	// 如果用户没有存储杂凑值且当前使用的是明文密码，为其计算并存储
	if user.PasswordHash == "" && req.Password != "" {
		newHash := models.CalculatePasswordHash(req.Password)
		models.DB.Model(user).Updates(map[string]interface{}{
			"password_hash": newHash,
			"hash_verified": true,
		})
	}

	// 验证用户鉴别信息完整性
	// 如果IntegrityHash为空（早期创建的用户），自动计算并更新
	if user.IntegrityHash == "" {
		user.IntegrityHash = models.CalculateUserIntegrityHash(user.Username, user.Email, user.HashedPassword)
		models.DB.Model(user).Updates(map[string]interface{}{"integrity_hash": user.IntegrityHash})
	} else if !user.VerifyIntegrity() {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户数据完整性验证失败",
			"code":  "INTEGRITY_CHECK_FAILED",
		})
		return
	}

	// 生成JWT令牌，包含用户ID、用户名和角色
	token, err := middleware.GenerateTokenWithUserIDAndRole(user.ID, user.Username, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败，请稍后重试"})
		return
	}

	// 返回令牌和用户信息
	// 计算用户完整权限列表 = 角色权限 + 自定义权限
	permissions := models.GetUserPermissions(string(user.Role), user.CustomPermissions)

	// 获取未读通知数量和最近的通知列表
	workflowService := services.NewWorkflowService(models.DB)
	unreadCount, _ := workflowService.GetUnreadNotificationCount(uint64(user.ID))
	notifications, _ := workflowService.GetUserNotifications(uint64(user.ID))
	var recentNotifications []models.Notification
	if len(notifications) > 5 {
		recentNotifications = notifications[:5]
	} else {
		recentNotifications = notifications
	}

	// 记录登录审计日志
	go func() {
		description := "用户 " + user.Username + " 登录成功"
		auditLog := models.AuditLog{
			UserID:     user.ID,
			Username:   user.Username,
			Action:     description,
			Module:     "认证",
			Method:     "POST",
			Path:       "/api/auth/login",
			IPAddress:  c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			StatusCode: http.StatusOK,
		}
		models.DB.Create(&auditLog)
	}()

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: token,
		ExpiresIn:   config.AppConfig.AccessTokenExpireMinutes * 60,
		TokenType:   "bearer",
		UserInfo: &UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			FullName:    user.FullName,
			Role:        string(user.Role),
			Permissions: permissions,
		},
		UnreadCount:   unreadCount,
		Notifications: recentNotifications,
	})
}

// GetUsers 获取用户列表处理器
// 返回分页的用户列表
// GET /api/auth/users
func (h *AuthHandler) GetUsers(c *gin.Context) {
	skip := 0   // 跳过记录数，默认从0开始
	limit := 20 // 每页数量，默认20条

	// 解析skip参数
	if s := c.Query("skip"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil && parsed >= 0 {
			skip = parsed
		}
	}

	// 解析limit参数，限制最大100条
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	username := c.Query("username")
	role := c.Query("role")

	// 调用服务层获取用户列表
	users, err := h.userService.GetUsers(skip, limit, username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	// 获取总数
	count, _ := h.userService.GetUsersCount(username, role)

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": count,
	})
}

// GetUserByID 根据ID获取用户详情处理器
// 返回指定用户的详细信息
// GET /api/auth/users/:user_id
func (h *AuthHandler) GetUserByID(c *gin.Context) {
	// 解析用户ID参数
	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	// 验证ID有效性
	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}

	// 调用服务层获取用户信息
	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser 更新用户信息处理器
// 修改用户的基本信息和角色
// PUT /api/auth/users/:user_id
func (h *AuthHandler) UpdateUser(c *gin.Context) {
	// 解析用户ID
	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}

	// 解析请求体
	var input services.UserUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式不正确"})
		return
	}

	// 验证并消毒邮箱
	if input.Email != "" {
		input.Email = sanitizeInput(input.Email)
		if err := validateEmail(input.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// 消毒其他可选字段
	if input.FullName != "" {
		input.FullName = sanitizeInput(input.FullName)
	}
	if input.Department != "" {
		input.Department = sanitizeInput(input.Department)
	}
	if input.Phone != "" {
		input.Phone = sanitizeInput(input.Phone)
	}

	// 如果要修改用户角色，需要管理员权限
	if input.Role != "" {
		currentUserID, _ := middleware.GetCurrentUserID(c)
		currentRole, _ := c.Get("role")

		// 只有管理员可以修改角色
		if currentRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权限修改用户角色"})
			return
		}

		// 防止管理员撤销自己的管理员权限
		if uint(id) == currentUserID && input.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能撤销自己的管理员权限"})
			return
		}
	}

	// 调用服务层更新用户
	user, err := h.userService.UpdateUser(uint(id), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户处理器
// 删除指定用户账号
// DELETE /api/auth/users/:user_id
func (h *AuthHandler) DeleteUser(c *gin.Context) {
	// 解析用户ID
	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}

	// 获取当前登录用户ID
	currentUserID, _ := middleware.GetCurrentUserID(c)

	// 防止删除自己的账号
	if uint(id) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账号"})
		return
	}

	// 调用服务层删除用户
	if err := h.userService.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RefreshToken 刷新访问令牌
// 使用当前有效的令牌获取新的访问令牌
// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从上下文中获取用户信息（AuthMiddleware已设置）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// 重新生成令牌
	token, err := middleware.GenerateTokenWithUserIDAndRole(
		userID.(uint),
		username.(string),
		role.(string),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌刷新失败"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: token,
		ExpiresIn:   config.AppConfig.AccessTokenExpireMinutes * 60,
		TokenType:   "bearer",
	})
}

// GetCurrentUserInfo 获取当前用户信息
// 返回当前登录用户的详细信息
// GET /api/auth/me
func (h *AuthHandler) GetCurrentUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	user, err := h.userService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 计算用户完整权限列表
	permissions := models.GetUserPermissions(string(user.Role), user.CustomPermissions)

	c.JSON(http.StatusOK, UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    user.FullName,
		Role:        string(user.Role),
		Permissions: permissions,
	})
}

// 默认重置密码
const DefaultResetPassword = "1qazXSW@"

// ResetPassword 重置用户密码
// 仅超级管理员可以操作，重置后返回新密码
// POST /api/auth/users/:user_id/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	// 检查是否为超级管理员
	currentRole, _ := c.Get("role")
	if currentRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以重置密码"})
		return
	}

	// 解析用户ID
	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID无效"})
		return
	}

	// 获取目标用户
	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 防止重置自己的密码
	currentUserID, _ := middleware.GetCurrentUserID(c)
	if uint(id) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能重置自己的密码"})
		return
	}

	// 重置密码
	newPassword := DefaultResetPassword
	if err := h.userService.ResetUserPassword(uint(id), newPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码重置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "密码重置成功",
		"username": user.Username,
		"password": newPassword,
	})
}

// UnlockUser 解锁用户账户
// 仅超级管理员可以操作，解锁后清除登录失败记录
// POST /api/auth/users/:user_id/unlock
func (h *AuthHandler) UnlockUser(c *gin.Context) {
	currentRole, _ := c.Get("role")
	if currentRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以解锁账户"})
		return
	}

	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	h.userService.ClearLoginFailures(user.Username)

	failCount, isLocked := h.userService.GetLoginFailureInfo(user.Username)
	c.JSON(http.StatusOK, gin.H{
		"message":    "解锁成功",
		"username":   user.Username,
		"fail_count": failCount,
		"is_locked":  isLocked,
	})
}

// GetUserLockStatus 获取用户账户锁定状态
// 仅超级管理员可以查看
// GET /api/auth/users/:user_id/lock-status
func (h *AuthHandler) GetUserLockStatus(c *gin.Context) {
	currentRole, _ := c.Get("role")
	if currentRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以查看锁定状态"})
		return
	}

	id, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式不正确"})
		return
	}

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	failCount, isLocked := h.userService.GetLoginFailureInfo(user.Username)
	c.JSON(http.StatusOK, gin.H{
		"username":     user.Username,
		"fail_count":   failCount,
		"is_locked":    isLocked,
		"max_attempts": models.MaxLoginAttempts,
	})
}
