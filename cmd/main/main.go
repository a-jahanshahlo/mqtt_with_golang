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
	app := router.Config{}
	router := gin.New()

	listenToMqtt()
	router.Use(app.Routes())
	router.GET("/ping", pong)

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
		fmt.Println("######Received message from " + msg.Channel + " channel.#####")
		fmt.Printf("%+v\n", warning)
	}
}

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
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())

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
