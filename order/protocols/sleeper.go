package protocols

import "time"

type Sleeper interface {
  Sleep (duration time.Duration)
}