package middleware

import (
	"contract-manage/config"
	"contract-manage/models"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// IssuerName JWT令牌发行者名称
// 用于标识该令牌是由本系统签发的
const (
	IssuerName = "anxin-contract-system"
)

// 全局数据库实例，用于权限检查
var permissionDB *gorm.DB

// SetPermissionDB 设置权限检查用的数据库实例
// 在main.go初始化时调用，避免循环导入
func SetPermissionDB(db *gorm.DB) {
	permissionDB = db
}

// Claims JWT声明结构
// 包含用户身份信息和标准JWT声明
type Claims struct {
	UserID               uint   `json:"user_id"` // 用户ID
	Username             string `json:"sub"`     // 用户名（Subject）
	Role                 string `json:"role"`    // 用户角色
	jwt.RegisteredClaims        // 标准JWT声明（过期时间、签发时间等）
}

// GenerateTokenWithUserID 生成JWT访问令牌（默认角色）
// 使用用户ID和用户名生成Token，默认角色为"user"
// 参数：userID-用户ID，username-用户名
// 返回：JWT令牌字符串和错误信息
func GenerateTokenWithUserID(userID uint, username string) (string, error) {
	// 计算令牌过期时间
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.AccessTokenExpireMinutes) * time.Minute)

	// 创建JWT声明
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     "user", // 默认角色
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),                  // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                      // 签发时间
			Issuer:    IssuerName,                                          // 发行者
			ID:        fmt.Sprintf("%d-%d", userID, time.Now().UnixNano()), // 唯一标识
		},
	}

	// 使用HS256算法签名，生成令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.SecretKey))
}

// GenerateTokenWithUserIDAndRole 生成JWT访问令牌（指定角色）
// 使用用户ID、用户名和角色生成Token
// 参数：userID-用户ID，username-用户名，role-用户角色
// 返回：JWT令牌字符串和错误信息
func GenerateTokenWithUserIDAndRole(userID uint, username string, role string) (string, error) {
	// 计算令牌过期时间
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.AccessTokenExpireMinutes) * time.Minute)

	// 创建JWT声明，包含用户角色
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    IssuerName,
			ID:        fmt.Sprintf("%d-%d", userID, time.Now().UnixNano()),
		},
	}

	// 使用HS256算法签名，生成令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.SecretKey))
}

// ParseToken 解析并验证JWT令牌
// 验证令牌签名、过期时间等信息的有效性
// 参数：tokenString- JWT令牌字符串
// 返回：解析后的Claims结构体和错误信息
func ParseToken(tokenString string) (*Claims, error) {
	// 检查令牌是否为空
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token is empty")
	}

	// 解析令牌并验证签名
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.Issuer != IssuerName {
			return nil, errors.New("invalid token issuer")
		}
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, errors.New("token has expired")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// AuthMiddleware JWT认证中间件
// 用于验证请求中的JWT令牌有效性
// 验证通过后将用户信息（ID、用户名、角色）存入Gin上下文
// 返回：Gin处理函数
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取Authorization字段
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 缺少认证头，返回401错误
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 解析Authorization头，格式应为 "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// 提取令牌字符串并验证基本格式
		tokenString := strings.TrimSpace(parts[1])
		if len(tokenString) < 10 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// 解析并验证令牌
		claims, err := ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":  "Invalid or expired token",
				"detail": "Please login again",
			})
			c.Abort()
			return
		}

		// 将用户信息存入Gin上下文，后续处理函数可直接获取
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		// 加载用户完整权限列表（角色权限 + 自定义权限）
		// 注意：这里需要避免循环导入，所以使用全局变量方式获取DB
		permissions := GetUserPermissionsFromDB(claims.UserID, claims.Role)
		c.Set("permissions", permissions)

		c.Next()
	}
}

// GetUserPermissionsFromDB 从数据库获取用户完整权限列表
// 参数：userID - 用户ID, role - 用户角色
// 返回：合并后的权限列表
func GetUserPermissionsFromDB(userID uint, role string) []string {
	fmt.Printf("[DEBUG] GetUserPermissionsFromDB: userID=%d, role=%s\n", userID, role)
	if permissionDB == nil {
		perms := GetDefaultPermissions(role)
		fmt.Printf("[DEBUG] permissionDB is nil, using default permissions: %v\n", perms)
		return perms
	}

	var user models.User
	if err := permissionDB.First(&user, userID).Error; err != nil {
		fmt.Printf("[DEBUG] failed to find user: %v\n", err)
		return GetDefaultPermissions(role)
	}

	fmt.Printf("[DEBUG] DB user role: %s, custom permissions: %s\n", user.Role, user.CustomPermissions)
	perms := models.GetUserPermissions(string(user.Role), user.CustomPermissions)
	fmt.Printf("[DEBUG] final permissions: %v\n", perms)
	return perms
}

// GetDefaultPermissions 获取角色默认权限
// 参数：role - 角色标识
// 返回：该角色的默认权限列表
func GetDefaultPermissions(role string) []string {
	rolePermissions := models.GetRolePermissions(role)
	return rolePermissions
}

// AdminRequiredMiddleware 管理员权限检查中间件
// 用于保护需要管理员权限的API接口
// 返回：Gin处理函数
func AdminRequiredMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户角色
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			// 非管理员用户，返回403禁止访问
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetCurrentUserID 获取当前请求的用户ID
// 从Gin上下文中提取已认证用户的ID
// 参数：c- Gin上下文
// 返回：用户ID和是否成功获取
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GetCurrentUsername 获取当前请求的用户名
// 参数：c- Gin上下文
// 返回：用户名和是否成功获取
func GetCurrentUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}
	name, ok := username.(string)
	return name, ok
}

// GetCurrentUserRole 获取当前请求的用户角色
// 参数：c- Gin上下文
// 返回：用户角色和是否成功获取
func GetCurrentUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}

// HashPassword 使用bcrypt加密密码
// 参数：password-明文密码
// 返回：加密后的密码字符串和错误信息
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码是否正确
// 参数：password-明文密码，hash-加密后的密码
// 返回：密码是否匹配
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// RequirePermission 权限检查中间件
// 用于保护需要特定权限的API接口
// 参数：permission - 需要的权限标识
// 返回：Gin处理函数
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户权限列表
		permissionsVal, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问"})
			c.Abort()
			return
		}

		permissions, ok := permissionsVal.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限信息无效"})
			c.Abort()
			return
		}

		// 检查是否拥有所需权限
		hasPermission := false
		for _, p := range permissions {
			if p == "all" || p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问: " + permission})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 权限检查中间件（满足任一权限即可）
// 参数：permissions - 需要的权限列表（满足任一即可）
// 返回：Gin处理函数
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPermissionsVal, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问"})
			c.Abort()
			return
		}

		userPermissions, ok := userPermissionsVal.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限信息无效"})
			c.Abort()
			return
		}

		hasPermission := false
		for _, userPerm := range userPermissions {
			if userPerm == "all" {
				hasPermission = true
				break
			}
			for _, requiredPerm := range permissions {
				if userPerm == requiredPerm {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions 权限检查中间件（需满足所有权限）
// 参数：permissions - 需要的所有权限列表
// 返回：Gin处理函数
func RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPermissionsVal, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问"})
			c.Abort()
			return
		}

		userPermissions, ok := userPermissionsVal.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限信息无效"})
			c.Abort()
			return
		}

		hasAllPermissions := true
		for _, requiredPerm := range permissions {
			found := false
			for _, userPerm := range userPermissions {
				if userPerm == "all" || userPerm == requiredPerm {
					found = true
					break
				}
			}
			if !found {
				hasAllPermissions = false
				break
			}
		}

		if !hasAllPermissions {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HasUserPermission 检查用户是否拥有指定权限
// 参数：userPermissions - 用户权限列表, requiredPermission - 需要的权限
// 返回：是否拥有权限
func HasUserPermission(userPermissions []string, requiredPermission string) bool {
	for _, p := range userPermissions {
		if p == "all" || p == requiredPermission {
			return true
		}
	}
	return false
}
