package raft

import (
	"math/rand"
	"time"
)

const minTimeout = 5000

func randomTimeout() <-chan time.Time {
	extra := time.Duration(rand.Int()) % minTimeout
	return time.After(minTimeout + extra)
}
