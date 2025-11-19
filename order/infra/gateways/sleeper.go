package gateways

import "time"

type Sleeper struct{}

func NewSleeper() *Sleeper {
	return &Sleeper{}
}

func (s *Sleeper) Sleep(duration time.Duration) {
	time.Sleep(duration)
}