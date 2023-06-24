package main

import (
	router "carriot/pkg/routes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
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

func main() {

	fmt.Println("Starting application...")
	//Open a goroutine execution start program
	//go manager.start()

	app := router.Config{}
	router := gin.New()

	listenToMqtt()
	router.Use(app.Routes())
	router.GET("/ping", pong)
	router.GET("/monitoring", wsHandler)
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
		jsonMessage, _ := json.Marshal(&Message{Sender: "ali-server", Content: string(msg.Payload), ServerIP: "sss", SenderIP: ""})
		manager.broadcast <- jsonMessage

		fmt.Println("**********Received message from " + msg.Channel + " *************")
		fmt.Printf("%+v\n", warning)
	}
}

// websocket
// Client management
type ClientManager struct {
	//The client map stores and manages all long connection clients, online is TRUE, and those who are not there are FALSE
	clients map[*Client]bool
	//Web side MESSAGE we use Broadcast to receive, and finally distribute it to all clients
	broadcast chan []byte
	//Newly created long connection client
	register chan *Client
	//Newly canceled long connection client
	unregister chan *Client
}

// Client
type Client struct {
	//User ID
	id string
	//Connected socket
	socket *websocket.Conn
	//Message
	send chan []byte
}

// Will formatting Message into JSON
type Message struct {
	//Message Struct
	Sender    string `json:"sender,omitempty"`    //发送者
	Recipient string `json:"recipient,omitempty"` //接收者
	Content   string `json:"content,omitempty"`   //内容
	ServerIP  string `json:"serverIp,omitempty"`  //实际不需要 验证k8s
	SenderIP  string `json:"senderIp,omitempty"`  //实际不需要 验证k8s
}

// Create a client manager
var manager = ClientManager{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

func (manager *ClientManager) start() {
	for {
		select {
		//If there is a new connection access, pass the connection to conn through the channel
		case conn := <-manager.register:
			//Set the client connection to true
			manager.clients[conn] = true
			//Format the message of returning to the successful connection JSON
			jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected. ", ServerIP: LocalIp(), SenderIP: conn.socket.RemoteAddr().String()})
			//Call the client's send method and send messages
			manager.send(jsonMessage, conn)
			//If the connection is disconnected
		case conn := <-manager.unregister:
			//Determine the state of the connection, if it is true, turn off Send and delete the value of connecting client
			if _, ok := manager.clients[conn]; ok {
				close(conn.send)
				delete(manager.clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected. ", ServerIP: LocalIp(), SenderIP: conn.socket.RemoteAddr().String()})
				manager.send(jsonMessage, conn)
			}
			//broadcast
		case message := <-manager.broadcast:
			//Traversing the client that has been connected, send the message to them
			for conn := range manager.clients {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(manager.clients, conn)
				}
			}
		}
	}
}

// Define the send method of client management
func (manager *ClientManager) send(message []byte, ignore *Client) {
	for conn := range manager.clients {
		//Send messages not to the shielded connection
		if conn != ignore {
			conn.send <- message
		}
	}
}

// Define the read method of the client structure
func (c *Client) read() {
	defer func() {
		manager.unregister <- c
		_ = c.socket.Close()
	}()

	for {
		//Read message
		_, message, err := c.socket.ReadMessage()
		//If there is an error message, cancel this connection and then close it
		if err != nil {
			manager.unregister <- c
			_ = c.socket.Close()
			break
		}
		//If there is no error message, put the information in Broadcast
		jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message), ServerIP: LocalIp(), SenderIP: c.socket.RemoteAddr().String()})
		manager.broadcast <- jsonMessage

		fmt.Println("**********Received message from   *************")
		fmt.Println(jsonMessage)
	}
}

func (c *Client) write() {
	defer func() {
		_ = c.socket.Close()
	}()

	for {
		select {
		//Read the message from send
		case message, ok := <-c.send:
			//If there is no message
			if !ok {
				_ = c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return

			}

			//Write it if there is news and send it to the web side
			_ = c.socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024 * 1024,
	WriteBufferSize: 1024 * 1024 * 1024,
	//Solving cross-domain problems
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(c *gin.Context) {

	//Upgrade the HTTP protocol to the websocket protocol
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		http.NotFound(c.Writer, c.Request)
		return
	}

	//Every connection will open a new client, client.id generates through UUID to ensure that each time it is different
	client := &Client{id: uuid.Must(uuid.NewV4(), nil).String(), socket: conn, send: make(chan []byte)}
	//Register a new link
	manager.register <- client

	//Start the message to collect the news from the web side
	go client.read()
	//Start the corporation to return the message to the web side
	go client.write()
}

func healthHandler(res http.ResponseWriter, _ *http.Request) {
	_, _ = res.Write([]byte("ok"))
}

func LocalIp() string {
	address, _ := net.InterfaceAddrs()
	var ip = "localhost"
	for _, address := range address {
		if ipAddress, ok := address.(*net.IPNet); ok && !ipAddress.IP.IsLoopback() {
			if ipAddress.IP.To4() != nil {
				ip = ipAddress.IP.String()
			}
		}
	}
	return ip
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
