package api

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/zhibo/backend/internal/api/handler"
	"github.com/zhibo/backend/internal/api/middleware"
	"github.com/zhibo/backend/internal/config"
	"github.com/zhibo/backend/internal/infra/kafka"
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
	var rdb *redisc.Client
	if opened, err := redisc.Open(cfg); err != nil {
		log.Printf("redis: %v (出价分布式锁已禁用，仅 DB 行锁+乐观锁)", err)
	} else {
		rdb = opened
		bidLocker = rdb
		roomCache = service.NewRedisRoomCache(rdb, sessionRepo)
		log.Printf("redis: connected %s (lock + room cache + event buffer)", cfg.RedisAddr)
	}

	var roomMQ ws.RoomBroadcaster
	if len(cfg.KafkaBrokers) > 0 {
		kb, err := kafka.NewRoomBroadcaster(cfg)
		if err != nil {
			log.Printf("kafka: %v (ws 将降级 redis pub/sub 或内存广播)", err)
		} else {
			roomMQ = kb
			log.Printf("kafka: brokers=%v topic=%s instance=%s", cfg.KafkaBrokers, cfg.KafkaTopic, cfg.InstanceID)
		}
	}
	auctionSvc := service.NewAuctionService(productRepo, sessionRepo, bidRepo, orderSvc)
	auctionSvc.SetSessionLocker(bidLocker)
	liveRoomRepo := repository.NewLiveRoomRepo(db)
	liveRoomSvc := service.NewLiveRoomService(liveRoomRepo, sessionRepo, productRepo, orderRepo, auctionSvc)
	bidSvc := service.NewBidService(db, sessionRepo, bidRepo, productRepo, orderRepo, bidLocker)
	if roomCache != nil {
		userAuctionSvc.SetRoomCache(roomCache)
		bidSvc.SetRoomCache(roomCache)
		auctionSvc.SetRoomCache(roomCache)
		liveRoomSvc.SetRoomCache(roomCache)
	}

	hub := ws.NewHub(sessionRepo, bidRepo, userAuctionSvc, bidSvc, rdb, roomMQ)
	liveRoomSvc.SetRoomViewerCounter(hub)
	wsNotifier := ws.NewNotifier(hub, bidRepo)
	if roomCache != nil {
		wsNotifier.SetRoomCache(roomCache)
	}
	messageSvc := service.NewMessageService(messageRepo, bidRepo)
	orderSvc.SetMessageService(messageSvc)
	roomNotifier := service.NewCompositeRoomNotifier(wsNotifier, messageSvc)
	bidSvc.SetRoomNotifier(roomNotifier)
	auctionSvc.SetRoomNotifier(roomNotifier)
	liveRoomSvc.SetRoomNotifier(roomNotifier)
	go auctionSvc.RunSettlementWorker(context.Background())

	metricsH := handler.NewMetricsHandler(hub)
	r.GET("/api/v1/metrics", metricsH.Get)
	r.GET("/metrics", metricsH.Prometheus)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authH := handler.NewAuthHandler(authSvc)

	wsH := ws.NewHandler(hub, userRepo, cfg.JWTSecret)
	r.GET("/api/v1/ws", wsH.ServeWS)

	aiSvc := service.NewAIService(cfg)
	ttsSvc := service.NewTTSService(cfg)
	aiH := handler.NewAIHandler(aiSvc, ttsSvc)

	productH := handler.NewProductHandler(productSvc)
	auctionH := handler.NewAuctionHandler(auctionSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	userAuctionH := handler.NewUserAuctionHandler(userAuctionSvc, bidSvc)
	userOrderH := handler.NewUserOrderHandler(orderSvc)
	messageH := handler.NewMessageHandler(messageSvc)
	streamH := handler.NewStreamHandler(cfg)
	liveRoomH := handler.NewLiveRoomHandler(liveRoomSvc)

	uploadDir := cfg.UploadDir
	if err := os.MkdirAll(filepath.Join(uploadDir, "products"), 0o755); err != nil {
		log.Printf("upload dir: %v", err)
	}
	r.Static("/uploads", uploadDir)
	uploadH := handler.NewUploadHandler(uploadDir)

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
		user.GET("/rooms/:roomId", liveRoomH.GetByRoom)
		user.GET("/rooms/:roomId/snapshot", userAuctionH.SnapshotByRoom)
		user.GET("/streams/:roomId", streamH.GetByRoom)
		user.POST("/tts", aiH.SynthesizeSpeech)

		user.POST("/auctions/:sessionId/bids", middleware.RequireAuth(userRepo, cfg.JWTSecret), userAuctionH.PlaceBid)

		userAuth := user.Group("")
		userAuth.Use(middleware.RequireAuth(userRepo, cfg.JWTSecret))
		{
			userAuth.GET("/orders", userOrderH.List)
			userAuth.GET("/orders/:id", userOrderH.Get)
			userAuth.GET("/auctions/:sessionId/order", userOrderH.GetBySession)
			userAuth.POST("/orders/:id/mock-pay", userOrderH.MockPay)
			userAuth.PUT("/orders/:id/shipping-address", userOrderH.SubmitShippingAddress)
			userAuth.POST("/orders/:id/confirm-receive", userOrderH.ConfirmReceive)
			userAuth.POST("/orders/:id/cancel", userOrderH.Cancel)

			userAuth.GET("/messages", messageH.List)
			userAuth.GET("/messages/unread-count", messageH.UnreadCount)
			userAuth.POST("/messages/:id/read", messageH.MarkRead)
			userAuth.POST("/messages/read-all", messageH.MarkAllRead)
		}
	}

	admin := v1.Group("/admin")
	admin.Use(middleware.RequireAuth(userRepo, cfg.JWTSecret), middleware.RequireAnchor())
	{
		admin.POST("/upload", uploadH.UploadImage)
		admin.POST("/products/ai-intro", aiH.GenerateProductIntro)
		admin.POST("/products", productH.Create)
		admin.GET("/products", productH.List)
		admin.GET("/products/:id", productH.Get)
		admin.PUT("/products/:id", productH.Update)
		admin.DELETE("/products/:id", productH.Delete)
		admin.POST("/products/:id/auctions", auctionH.Publish)

		admin.GET("/auctions/:sessionId", auctionH.Get)
		admin.PUT("/auctions/:sessionId/rules", auctionH.UpdateRules)
		admin.POST("/auctions/:sessionId/cancel", auctionH.Cancel)

		admin.POST("/live-rooms", liveRoomH.Create)
		admin.GET("/live-rooms", liveRoomH.List)
		admin.GET("/live-rooms/:id", liveRoomH.Get)
		admin.POST("/live-rooms/:id/start", liveRoomH.Start)
		admin.POST("/live-rooms/:id/end", liveRoomH.End)
		admin.POST("/live-rooms/:id/sessions", liveRoomH.AddSession)
		admin.POST("/live-rooms/:id/end-current", liveRoomH.EndCurrentAndSwitch)

		admin.GET("/orders", orderH.List)
		admin.GET("/orders/:id", orderH.Get)
		admin.POST("/orders/:id/ship", orderH.Ship)
		admin.POST("/orders/:id/cancel", orderH.Cancel)
		admin.POST("/orders/:id/refund", orderH.Refund)
	}

	return r
}
