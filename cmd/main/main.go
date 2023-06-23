package main

import (
	router "carriot/pkg/routes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
)

type HasWarnings struct {
	DeviceID    string    `json:"deviceID"`
	WarningTime time.Time `json:"warningTime"`
	WarningType int       `json:"warningType"`
}
type TempLog struct {
	DeviceID       string    `json:"deviceID"`
	DeviceTime     time.Time `json:"deviceTime"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Altitude       float64   `json:"altitude"`
	Course         float64   `json:"course"`
	Satellites     int       `json:"satellites"`
	SpeedOTG       float32   `json:"speedOTG"`
	AccelerationX1 float32   `json:"accelerationX1"`
	AccelerationY1 float32   `json:"accelerationY1"`
	Signal         int       `json:"signal"`
	PowerSupply    int       `json:"powerSupply"`
}
type connection struct {
	ws   *websocket.Conn
	send chan []byte
}
type hub struct {
	rooms      map[string]map[*connection]bool
	broadcast  chan message
	register   chan subscription
	unregister chan subscription
}
type subscription struct {
	conn *connection
}
type message struct {
	data []byte
	room string
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var h = hub{
	broadcast: make(chan message),
	rooms:     make(map[string]map[*connection]bool),
}

func main() {
	app := router.Config{}
	router := gin.New()
	go h.run()

	listenToMqtt()
	router.Use(app.Routes())
	router.GET("/ping", pong)
	router.GET("/monitoring", pong)
	s := &http.Server{
		Addr:           ":8087",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
	//redis

	subscriber := redisClient.Subscribe("detection_queue")
	defer subscriber.Close()
	var warning = HasWarnings{}
	for {
		msg, err := subscriber.ReceiveMessage()
		if err != nil {
			panic(err)
		}

		if err := json.Unmarshal([]byte(msg.Payload), &warning); err != nil {
			panic(err)
		}
		fmt.Println("**********Received message from " + msg.Channel + " *************")
		fmt.Printf("%+v\n", warning)
	}
}

// websocket
func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err.Error())
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws}
	s := subscription{c}
	h.register <- s
	go s.writePump()
	go s.readPump()
}

func (s subscription) readPump() {
	c := s.conn
	defer func() {
		h.unregister <- s
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		m := message{msg, "1"}
		h.broadcast <- m
	}
}
func (s *subscription) writePump() {
	c := s.conn
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (h *hub) run() {
	for {
		select {

		case m := <-h.broadcast:
			connections := h.rooms[m.room]
			for c := range connections {
				select {
				case c.send <- m.data:
				default:
					close(c.send)
					delete(connections, c)
					if len(connections) == 0 {
						delete(h.rooms, m.room)
					}
				}
			}
		}
	}
}

// h
func pong(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"hello": "world"})
}
func publishToRedis(log TempLog) {
	fmt.Printf("Start to pub redis############:")
	payload, err := json.Marshal(log)
	if err != nil {
		panic(err)
	}

	if err := redisClient.Publish("has_warnings_queue", payload).Err(); err != nil {
		panic(err)
	}
	fmt.Printf("End to pub redis$$$$$$$$$$$$$$$$$")

}

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	//fmt.Printf("TOPIC: %s\n", msg.Topic())
	//fmt.Printf("MSG: %s\n", msg.Payload())

	var log = TempLog{}
	if err := json.Unmarshal([]byte(msg.Payload()), &log); err != nil {
		panic(err)
	}
	publishToRedis(log)
}

func listenToMqtt() {
	mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	opts := mqtt.NewClientOptions().AddBroker("tcp://79.175.157.228:1883").SetUsername("interview").SetPassword("interview123")

	opts.SetKeepAlive(60 * time.Second)
	// Set the message callback handler
	opts.SetDefaultPublishHandler(f)
	opts.SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	token := c.Connect()
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to a topic
	if token := c.Subscribe("sensor_logs/#", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Publish a message

	time.Sleep(6 * time.Second)

	// Disconnect

}

var ctx = context.Background()

var redisClient = redis.NewClient(&redis.Options{
	Addr:     "79.175.157.233:6379",
	Password: "asb31cdnaksord",
})
