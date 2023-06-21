package main

import "time"

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

type HasWarnings struct {
	DeviceID    string    `json:"deviceID"`
	WarningTime time.Time `json:"warningTime"`
	WarningType int       `json:"warningType"`
}
