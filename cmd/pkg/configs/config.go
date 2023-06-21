package main

type MqttConfig struct {
	Address  string
	Port     int
	Username string
	Password string
	Topic    string
}
type RedisConfig struct {
	Address  string
	Port     int
	Username string
	Password string
	Topic    string
}
type ConstsRedisChannel struct {
	HasWarningsQueueChannel string
	DetectionQueueChannel   string
}
