package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 令牌桶限流器
// 基于令牌桶算法实现对请求的限流控制
type RateLimiter struct {
	visitors map[string]*visitor // 访问者映射，key为IP地址
	mu       sync.RWMutex        // 读写锁，保护visitors map
	rate     int                 // 令牌恢复速率（每秒恢复的令牌数）
	burst    int                 // 令牌桶容量（初始令牌数）
	expires  time.Duration       // 访问记录过期时间
}

// visitor 访问者结构
// 记录每个IP的令牌数量和最后访问时间
type visitor struct {
	tokens    int       // 当前令牌数量
	lastVisit time.Time // 最后访问时间
}

// NewRateLimiter 创建新的限流器实例
// 参数：rate-每秒恢复令牌数，burst-初始令牌数，expires-记录过期时间
// 返回：限流器指针
func NewRateLimiter(rate int, burst int, expires time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		expires:  expires,
	}
	// 启动后台清理goroutine，定期清除过期记录
	go rl.cleanup()
	return rl
}

// getVisitor 获取或创建访问者记录
// 如果访问者不存在则创建新的记录
// 参数：ip-客户端IP地址
// 返回：访问者结构指针
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		// 新访问者，初始化令牌桶
		rl.visitors[ip] = &visitor{tokens: rl.burst, lastVisit: time.Now()}
		return rl.visitors[ip]
	}

	// 计算时间差，恢复令牌
	now := time.Now()
	elapsed := now.Sub(v.lastVisit)
	// 恢复的令牌数 = 经过的秒数 * 速率，上限为burst
	v.tokens = min(rl.burst, v.tokens+int(elapsed.Seconds()*float64(rl.rate)))
	v.lastVisit = now

	return v
}

// Allow 检查是否允许请求通过
// 基于令牌桶算法判断当前是否还有可用令牌
// 参数：ip-客户端IP地址
// 返回：true表示允许通过，false表示被限流
func (rl *RateLimiter) Allow(ip string) bool {
	v := rl.getVisitor(ip)
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

// cleanup 定期清理过期的访问记录
// 每隔expires时间检查并删除长期未活跃的IP记录
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.expires)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			// 如果超过expires时间未访问，则删除记录
			if now.Sub(v.lastVisit) > rl.expires {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// 全局限流器实例，使用sync.Once确保只初始化一次
var (
	rateLimiter     *RateLimiter
	rateLimiterOnce sync.Once
)

// GetRateLimiter 获取全局限流器实例
// 初始化配置：每秒60个令牌，初始100个令牌，记录1小时后过期
// 返回：限流器指针
func GetRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		rateLimiter = NewRateLimiter(60, 100, time.Hour)
	})
	return rateLimiter
}

// RateLimitMiddleware 请求限流中间件
// 使用令牌桶算法限制每个IP的请求频率
// 返回：Gin处理函数
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查客户端IP是否被限流
		if !GetRateLimiter().Allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// 注册限流器实例（更严格的限制）
var (
	registerRateLimiter     *RateLimiter
	registerRateLimiterOnce sync.Once
)

// GetRegisterRateLimiter 获取注册专用限流器
// 限制：每分钟最多3次注册请求
func GetRegisterRateLimiter() *RateLimiter {
	registerRateLimiterOnce.Do(func() {
		registerRateLimiter = NewRateLimiter(3, 3, time.Minute)
	})
	return registerRateLimiter
}

// RegisterRateLimitMiddleware 注册接口限流中间件
// 更严格的限制：每分钟最多3次注册请求
func RegisterRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !GetRegisterRateLimiter().Allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "注册请求过于频繁，请稍后再试",
				"retry_after": 60,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SecureHeadersMiddleware 安全响应头中间件
// 设置多种安全相关的HTTP响应头，防止常见Web攻击
// 返回：Gin处理函数
func SecureHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止MIME类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")
		// 防止页面被iframe嵌入（防止点击劫持）
		c.Header("X-Frame-Options", "DENY")
		// 启用XSS防护
		c.Header("X-XSS-Protection", "1; mode=block")
		// 强制HTTPS连接
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")
		// 引用来源策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// 浏览器功能策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

// CORSMiddleware 跨域资源共享中间件
// 处理浏览器的跨域请求，支持预检请求
// 返回：Gin处理函数
func CORSMiddleware() gin.HandlerFunc {
	// 允许的来源列表 - 生产环境应该配置精确的域名
	isProduction := os.Getenv("GIN_MODE") == "release"

	var allowedOrigins []string
	if isProduction {
		// 生产环境：从环境变量读取允许的域名
		allowedOrigins = []string{}
	} else {
		// 开发环境：允许本地开发
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
		}
	}

	// 允许所有localhost（开发用）
	allowAllLocalhost := !isProduction

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 检查来源是否在允许列表中
		allowed := false
		if allowAllLocalhost && strings.HasPrefix(origin, "http://localhost:") {
			allowed = true
		}
		if !allowed {
			for _, o := range allowedOrigins {
				if origin == o {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// 设置允许的HTTP方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 设置允许的请求头
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		// 允许携带认证信息
		c.Header("Access-Control-Allow-Credentials", "true")
		// 预检请求缓存时间
		c.Header("Access-Control-Max-Age", "3600")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestSizeLimitMiddleware 请求体大小限制中间件
// 限制客户端请求体的最大大小，防止大文件上传攻击
// 参数：maxSize-允许的最大字节数
// 返回：Gin处理函数
func RequestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "请求体过大",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
