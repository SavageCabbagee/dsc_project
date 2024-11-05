package raft

import "sync"

type Log struct {
	Index   int
	Term    int
	Command Command
}

type RaftLogStore struct {
	logs []Log
	lock *sync.RWMutex
}

func (logStore *RaftLogStore) GetLastLogIndex() int {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	return len(logStore.logs)
}

func (logStore *RaftLogStore) GetLastLogTerm() int {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	if len(logStore.logs) == 0 {
		return 0
	}
	return logStore.logs[len(logStore.logs)-1].Term
}

func (logStore *RaftLogStore) GetLog(index int) Log {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	if index-1 < 0 {
		return Log{0, 0, Command{}}
	}
	return logStore.logs[index-1]
}

func (logStore *RaftLogStore) GetLogRange(startIndex, endIndex int) []Log {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	return logStore.logs[startIndex-1 : endIndex-1]
}

func (logStore *RaftLogStore) GetLogsFrom(startIndex int) []Log {
	logStore.lock.RLock()
	defer logStore.lock.RUnlock()
	start := startIndex - 1
	if start < 0 {
		start = 0
	}
	return logStore.logs[start:]
}

func (logStore *RaftLogStore) AppendLog(command Command, term int) {
	logStore.lock.Lock()
	defer logStore.lock.Unlock()
	logStore.logs = append(logStore.logs, Log{len(logStore.logs) + 1, term, command})
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
			if entry.Term != logStore.logs[entry.Index-1].Term {
				logStore.logs = logStore.logs[:entry.Index]
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
