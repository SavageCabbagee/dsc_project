package raft

import (
	"sync"
	"time"
)

type StableState struct {
	currentTerm int
	votedFor    int
	lastContact time.Time

	lock sync.RWMutex
}

func (state *StableState) GetCurrentTerm() int {
	state.lock.RLock()
	defer state.lock.RUnlock()
	return state.currentTerm
}

func (state *StableState) SetCurrentTerm(newTerm int) {
	state.lock.Lock()
	defer state.lock.Unlock()
	state.currentTerm = newTerm
}

func (state *StableState) GetVotedFor() int {
	state.lock.RLock()
	defer state.lock.RUnlock()
	return state.votedFor
}

func (state *StableState) SetVotedFor(newVote int) {
	state.lock.Lock()
	defer state.lock.Unlock()
	state.votedFor = newVote
}

func (state *StableState) GetLastContact() time.Time {
	state.lock.RLock()
	defer state.lock.RUnlock()
	return state.lastContact
}

func (state *StableState) SetLastContact(newContact time.Time) {
	state.lock.Lock()
	defer state.lock.Unlock()
	state.lastContact = newContact
}
