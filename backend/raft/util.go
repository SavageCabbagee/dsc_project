package raft

import (
	"fmt"
	"math/rand"
	"time"
)

const minTimeout = 5 * time.Second
const leaderHeartbeat = 2 * time.Second

func randomTimeout() <-chan time.Time {
	extra := time.Duration(rand.Int()) % minTimeout
	fmt.Println(minTimeout + extra)
	return time.After(minTimeout + extra)
}

func leaderHeartbeatTimeout() <-chan time.Time {
	return time.After(leaderHeartbeat)
}
