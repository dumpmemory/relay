package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DGHeroin/relay/webui/model"
	"github.com/gin-gonic/gin"
)

// CORS 设置缓存
var (
	corsCache struct {
		sync.RWMutex
		origin    string
		updatedAt time.Time
	}
	corsCacheTTL = 30 * time.Second // 缓存 30 秒
)

//go:embed all:frontend/dist
var frontendFS embed.FS

// Server Web服务器
type Server struct {
	engine   *gin.Engine
	addr     string
	handlers *Handlers
}

// NewServer 创建服务器
func NewServer(addr string) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())

	s := &Server{
		engine:   engine,
		addr:     addr,
		handlers: NewHandlers(),
	}

	s.setupRoutes()
	return s
}

// getCachedCORSOrigin 获取缓存的 CORS 来源设置
func getCachedCORSOrigin() string {
	corsCache.RLock()
	if time.Since(corsCache.updatedAt) < corsCacheTTL {
		origin := corsCache.origin
		corsCache.RUnlock()
		return origin
	}
	corsCache.RUnlock()

	// 缓存过期，重新获取
	corsCache.Lock()
	defer corsCache.Unlock()

	// 双重检查，避免多个 goroutine 同时刷新
	if time.Since(corsCache.updatedAt) < corsCacheTTL {
		return corsCache.origin
	}

	allowOrigin, err := model.GetSetting("cors_origin")
	if err != nil || allowOrigin == "" {
		// 默认不允许跨域（同源策略）
		// 如需允许跨域，请在设置中配置 cors_origin
		allowOrigin = ""
	}
	corsCache.origin = allowOrigin
	corsCache.updatedAt = time.Now()
	return allowOrigin
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// 使用缓存的 CORS 设置
		allowOrigin := getCachedCORSOrigin()

		// 检查是否允许该来源
		allowed := false
		if allowOrigin == "*" {
			allowed = true
		} else {
			// 支持多个来源，用逗号分隔
			origins := strings.Split(allowOrigin, ",")
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (s *Server) setupRoutes() {
	// 健康检查（无需鉴权）
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"version":      Version,
			"need_setup":   !model.IsSetupCompleted(),
		})
	})

	// 统一 API 入口
	s.engine.POST("/api", s.handleAPI)

	// WebSocket
	s.engine.GET("/ws", s.handlers.HandleWebSocket)

	// GeoIP 文件上传 (multipart/form-data)
	s.engine.POST("/api/upload/geoip", s.handlers.HandleGeoIPUpload)

	// 静态文件
	s.setupStaticFiles()
}

func (s *Server) setupStaticFiles() {
	// 尝试使用嵌入的前端文件
	subFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("警告: 未找到嵌入的前端文件，使用开发模式")
		// 开发模式：使用本地文件
		s.engine.Static("/assets", "./webui/frontend/dist/assets")
		s.engine.StaticFile("/", "./webui/frontend/dist/index.html")
		s.engine.NoRoute(func(c *gin.Context) {
			c.File("./webui/frontend/dist/index.html")
		})
		return
	}

	// 生产模式：使用嵌入文件
	s.engine.StaticFS("/assets", http.FS(mustSubFS(subFS, "assets")))

	// SPA 路由支持
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// API 和 WebSocket 请求不走静态文件
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") {
			c.JSON(404, gin.H{"code": 404, "msg": "not found"})
			return
		}
		// 返回 index.html
		data, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			c.String(500, "Internal Server Error")
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})
}

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		return fsys
	}
	return sub
}

// handleAPI 统一 API 处理
func (s *Server) handleAPI(c *gin.Context) {
	var req APIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, APIResponse{Code: 400, Msg: "请求格式错误"})
		return
	}

	resp := s.handlers.Handle(req.Action, req.Data, c)
	c.JSON(200, resp)
}

// Run 启动服务器
func (s *Server) Run() error {
	log.Printf("服务器启动: http://%s", s.addr)
	return s.engine.Run(s.addr)
}

// APIRequest 统一请求格式
type APIRequest struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}

// APIResponse 统一响应格式
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(data interface{}) APIResponse {
	return APIResponse{Code: 0, Msg: "success", Data: data}
}

func Error(code int, msg string) APIResponse {
	return APIResponse{Code: code, Msg: msg}
}
