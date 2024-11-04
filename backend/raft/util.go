package raft

import (
	"math/rand"
	"time"
)

const TIMEOUT = 5000

func randomTimeout() <-chan time.Time {
	extra := time.Duration(rand.Int()) % TIMEOUT
	return time.After(TIMEOUT + extra)
}
