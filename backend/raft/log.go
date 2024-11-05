package raft

import "sync"

type Log struct {
	Index   int
	Term    int
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

func (logStore *RaftLogStore) AppendEntries(logs []Log) {
	logStore.lock.Lock()
	defer logStore.lock.Unlock()
	if len(logs) > 0 {
		var newEntries []Log
		for i, entry := range logs {
			if entry.Index > len(logStore.logs) {
				newEntries = logs[i:]
				break
			}
			if entry.Term != logStore.logs[entry.Index].Term {
				logStore.logs = logStore.logs[:entry.Index+1]
				newEntries = logs[i:]
				break
			}
		}

		if n := len(newEntries); n > 0 {
			// Append the new entries
			logStore.logs = append(logStore.logs, newEntries...)
		}
	}
}
