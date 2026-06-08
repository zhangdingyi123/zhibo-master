package api

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/handler"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/config"
	redisc "github.com/zhibo/backend/internal/infra/redis"
	"github.com/zhibo/backend/internal/repository"
	"github.com/zhibo/backend/internal/service"
	"github.com/zhibo/backend/internal/ws"
)

func NewRouter(cfg config.Config, db *sql.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendURLs,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Mock-Open-Id", "X-User-Id", "X-Client-Id"},
		AllowCredentials: true,
	}))

	health := handler.NewHealthHandler()
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "zhibo-api",
			"hint":    "这是 API 服务，请访问前端 http://localhost:5173 或接口 /api/v1/health",
			"health":  "/api/v1/health",
			"docs":    "/api/v1/ping",
		})
	})
	r.GET("/health", health.Check)
	r.GET("/api/v1/health", health.Check)

	userRepo := repository.NewUserRepo(db)
	productRepo := repository.NewProductRepo(db)
	sessionRepo := repository.NewSessionRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	bidRepo := repository.NewBidRepo(db)
	messageRepo := repository.NewMessageRepo(db)

	payTimeout := time.Duration(cfg.PayTimeoutMinutes) * time.Minute
	orderSvc := service.NewOrderService(orderRepo, productRepo, payTimeout)
	go orderSvc.RunPayExpiryWorker(context.Background())
	productSvc := service.NewProductService(productRepo, sessionRepo, orderRepo)
	userAuctionSvc := service.NewUserAuctionService(sessionRepo, productRepo)
	var bidLocker service.SessionLocker = service.NoopLocker{}
	var roomCache service.RoomCache
	if rdb, err := redisc.Open(cfg); err != nil {
		log.Printf("redis: %v (出价分布式锁已禁用，仅 DB 行锁+乐观锁)", err)
	} else {
		bidLocker = rdb
		roomCache = service.NewRedisRoomCache(rdb, sessionRepo)
		log.Printf("redis: connected %s (lock + room cache)", cfg.RedisAddr)
	}
	auctionSvc := service.NewAuctionService(productRepo, sessionRepo, bidRepo, orderSvc)
	auctionSvc.SetSessionLocker(bidLocker)
	bidSvc := service.NewBidService(db, sessionRepo, bidRepo, productRepo, orderRepo, bidLocker)
	if roomCache != nil {
		userAuctionSvc.SetRoomCache(roomCache)
		bidSvc.SetRoomCache(roomCache)
		auctionSvc.SetRoomCache(roomCache)
	}

	hub := ws.NewHub(sessionRepo, bidRepo, userAuctionSvc, bidSvc)
	wsNotifier := ws.NewNotifier(hub, bidRepo)
	if roomCache != nil {
		wsNotifier.SetRoomCache(roomCache)
	}
	messageSvc := service.NewMessageService(messageRepo, bidRepo)
	roomNotifier := service.NewCompositeRoomNotifier(wsNotifier, messageSvc)
	bidSvc.SetRoomNotifier(roomNotifier)
	auctionSvc.SetRoomNotifier(roomNotifier)
	go auctionSvc.RunSettlementWorker(context.Background())

	metricsH := handler.NewMetricsHandler(hub)
	r.GET("/api/v1/metrics", metricsH.Get)
	r.GET("/metrics", metricsH.Prometheus)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authH := handler.NewAuthHandler(authSvc)

	wsH := ws.NewHandler(hub, userRepo, cfg.JWTSecret)
	r.GET("/api/v1/ws", wsH.ServeWS)

	productH := handler.NewProductHandler(productSvc)
	auctionH := handler.NewAuctionHandler(auctionSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	userAuctionH := handler.NewUserAuctionHandler(userAuctionSvc, bidSvc)
	userOrderH := handler.NewUserOrderHandler(orderSvc)
	messageH := handler.NewMessageHandler(messageSvc)
	streamH := handler.NewStreamHandler(cfg)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})
		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)
		v1.GET("/auth/me", middleware.RequireAuth(userRepo, cfg.JWTSecret), authH.Me)
	}

	// 用户端（2.7–2.9）：列表/详情/快照可匿名；出价需登录
	user := v1.Group("")
	{
		user.GET("/auctions", userAuctionH.List)
		user.GET("/auctions/:sessionId", userAuctionH.Get)
		user.GET("/auctions/:sessionId/snapshot", userAuctionH.Snapshot)
		user.GET("/rooms/:roomId/snapshot", userAuctionH.SnapshotByRoom)
		user.GET("/streams/:roomId", streamH.GetByRoom)

		user.POST("/auctions/:sessionId/bids", middleware.RequireAuth(userRepo, cfg.JWTSecret), userAuctionH.PlaceBid)

		userAuth := user.Group("")
		userAuth.Use(middleware.RequireAuth(userRepo, cfg.JWTSecret))
		{
			userAuth.GET("/orders", userOrderH.List)
			userAuth.GET("/orders/:id", userOrderH.Get)
			userAuth.GET("/auctions/:sessionId/order", userOrderH.GetBySession)
			userAuth.POST("/orders/:id/mock-pay", userOrderH.MockPay)

			userAuth.GET("/messages", messageH.List)
			userAuth.GET("/messages/unread-count", messageH.UnreadCount)
			userAuth.POST("/messages/:id/read", messageH.MarkRead)
			userAuth.POST("/messages/read-all", messageH.MarkAllRead)
		}
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.RequireAuth(userRepo, cfg.JWTSecret), middleware.RequireAnchor())
	{
		admin.POST("/products", productH.Create)
		admin.GET("/products", productH.List)
		admin.GET("/products/:id", productH.Get)
		admin.PUT("/products/:id", productH.Update)
		admin.DELETE("/products/:id", productH.Delete)
		admin.POST("/products/:id/auctions", auctionH.Publish)

		admin.GET("/auctions/:sessionId", auctionH.Get)
		admin.PUT("/auctions/:sessionId/rules", auctionH.UpdateRules)
		admin.POST("/auctions/:sessionId/cancel", auctionH.Cancel)

		admin.GET("/orders", orderH.List)
		admin.GET("/orders/:id", orderH.Get)
	}

	return r
}
