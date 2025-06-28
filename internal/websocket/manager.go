package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене здесь должна быть более строгая проверка
		return true
	},
}

// Message представляет сообщение WebSocket
type Message struct {
	Type        string      `json:"type"`
	Data        interface{} `json:"data,omitempty"`
	CameraID    string      `json:"camera_id,omitempty"`
	CameraName  string      `json:"camera_name,omitempty"`
	Status      string      `json:"status,omitempty"`
	Event       interface{} `json:"event,omitempty"`
	Message     string      `json:"message,omitempty"`
	Level       string      `json:"level,omitempty"`
	ObjectClass string      `json:"object_class,omitempty"`
	Confidence  float64     `json:"confidence,omitempty"`
	Camera      interface{} `json:"camera,omitempty"`
}

// Client представляет WebSocket клиента
type Client struct {
	conn   *websocket.Conn
	send   chan *Message
	hub    *Hub
	userID string
}

// Hub управляет WebSocket подключениями
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mutex      sync.RWMutex
}

// NewHub создает новый WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

// Run запускает WebSocket hub
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket hub stopping...")
			return

		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket client connected, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()
			log.Printf("WebSocket client disconnected, total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Broadcast отправляет сообщение всем подключенным клиентам
func (h *Hub) Broadcast(message *Message) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("Warning: WebSocket broadcast channel is full")
	}
}

// GetClientCount возвращает количество подключенных клиентов
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// ServeWS обрабатывает WebSocket подключения
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Получаем userID из контекста (если есть аутентификация)
	userID := ""
	if user := r.Context().Value("user"); user != nil {
		if userMap, ok := user.(map[string]interface{}); ok {
			if id, ok := userMap["id"].(string); ok {
				userID = id
			}
		}
	}

	client := &Client{
		conn:   conn,
		send:   make(chan *Message, 256),
		hub:    h,
		userID: userID,
	}

	client.hub.register <- client

	// Запускаем горутины для чтения и записи
	go client.writePump()
	go client.readPump()
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// readPump обрабатывает входящие сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Обрабатываем входящие сообщения от клиента
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			log.Printf("Received WebSocket message: %s", msg.Type)
			// Здесь можно добавить обработку команд от клиента
		}
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
