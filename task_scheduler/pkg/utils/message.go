package utils

import (
	"time"
)

type Request struct {
	Input string 
}

type TimePack struct {
  Get         time.Time
  Issue       time.Time
  Finish      time.Time
  Response    time.Time
  Next *TimePack
}

type Response struct {
	Output string 
	Time  TimePack
}