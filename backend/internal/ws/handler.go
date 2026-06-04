package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zhibo/backend/internal/repository"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境；生产应校验 Origin
	},
}

// Handler WebSocket HTTP 升级入口
type Handler struct {
	hub       *Hub
	users     *repository.UserRepo
	jwtSecret string
}

func NewHandler(hub *Hub, users *repository.UserRepo, jwtSecret string) *Handler {
	return &Handler{hub: hub, users: users, jwtSecret: jwtSecret}
}

// ServeWS GET /api/v1/ws?roomId=...&clientId=...&lastSeq=0&token=JWT
func (h *Handler) ServeWS(c *gin.Context) {
	user, _ := resolveUser(c.Request, h.users, h.jwtSecret)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	clientID := c.Query("clientId")
	if clientID == "" {
		clientID = c.GetHeader("X-Client-Id")
	}
	if clientID == "" {
		clientID = generateClientID()
	}

	var userID uint64
	if user != nil {
		userID = user.ID
	}

	client := &Client{
		hub:      h.hub,
		conn:     conn,
		send:     make(chan []byte, sendBuffer),
		roomID:   c.Query("roomId"),
		clientID: clientID,
		userID:   userID,
	}

	go client.writePump()
	go client.readPump()

	// Query 参数可自动订阅
	if client.roomID != "" {
		lastSeq := parseUint64(c.Query("lastSeq"))
		h.hub.handleSubscribe(client, SubscribePayload{
			RoomID:   client.roomID,
			ClientID: client.clientID,
			LastSeq:  lastSeq,
		})
	}
}

func parseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + uint64(ch-'0')
	}
	return n
}

func generateClientID() string {
	return "c_" + randomHex(8)
}
