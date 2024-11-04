package raft

import "sync"

type Log struct {
	Index   uint64
	Term    uint64
	Command Command
}

type RaftLogStore struct {
	logs []Log
	lock sync.RWMutex
}

func (logStore *RaftLogStore) GetLog(index uint64) Log {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	return logStore.logs[index]
}
