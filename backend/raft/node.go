package raft

import (
	"net/rpc"
	"sync/atomic"
	"time"
)

const (
	FOLLOWER  int32 = 0
	CANDIDATE int32 = 1
	LEADER    int32 = 2
)

type RaftNode struct {
	id          int
	stableState StableState
	logStore    RaftLogStore

	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	store Store

	heartbeatTimer <-chan time.Time

	leaderAddr string
	leaderId   int

	state atomic.Int32
	peers map[int]*rpc.Client // key: id, string: client
}

func (node *RaftNode) run() {
	for {
		switch node.state.Load() {
		case FOLLOWER:
			node.runFollower()
		case CANDIDATE:
			node.runCandidate()
		case LEADER:
			node.runLeader()
		}
	}
}

func (node *RaftNode) runFollower() {
	for node.state.Load() == FOLLOWER {
		select {
		case <-node.heartbeatTimer:
			if time.Since(node.stableState.GetLastContact()) < minTimeout {
				continue
			}
			node.state.Store(CANDIDATE)
		default:
			continue
		}
	}
}
func (node *RaftNode) runCandidate() {
	node.stableState.SetCurrentTerm(node.stableState.GetCurrentTerm() + 1)
	node.stableState.SetVotedFor(node.id)
	node.resetTimer()
	// goroutine for sending requestvote
	// timeout
	for node.state.Load() == CANDIDATE {
	}

}
func (node *RaftNode) runLeader() {}

func (node *RaftNode) updateTerm(term int) {
	node.stableState.SetCurrentTerm(term)
	node.stableState.SetVotedFor(-1)
	node.state.Store(FOLLOWER)
}

func (node *RaftNode) resetTimer() {
	node.heartbeatTimer = randomTimeout()
}

func (node *RaftNode) handleAppendEntries(args *AppendEntriesArgs) (bool, int) {
	node.stableState.SetLastContact(time.Now())
	node.resetTimer()
	if args.term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.term)
	}

	// update leader
	node.leaderId = args.leaderId

	if len(node.logStore.logs) < args.prevLogIndex || node.logStore.logs[args.prevLogIndex].Term != args.term {
		return false, node.stableState.GetCurrentTerm()
	}
	// handle 3 and 4
	node.logStore.AppendEntries(args.logs)
	if args.leaderCommit > node.commitIndex {
		node.commitIndex = min(args.leaderCommit, len(node.logStore.logs))
		for node.lastApplied < node.commitIndex {
			node.store.ApplyLogs(node.logStore.logs[node.lastApplied+1 : node.commitIndex])
			node.lastApplied = node.commitIndex
		}
	}
	return true, node.stableState.GetCurrentTerm()
}

func (node *RaftNode) handleRequestVote(args *RequestVoteArgs) (bool, int) {
	if time.Since(node.stableState.GetLastContact()) < minTimeout {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.term)
	}

	if node.stableState.GetVotedFor() != -1 || node.stableState.GetVotedFor() != args.candidateId {
		return false, node.stableState.GetCurrentTerm()
	}

	if node.logStore.logs[len(node.logStore.logs)-1].Term > args.lastLogTerm ||
		len(node.logStore.logs) > args.lastLogIndex {
		return false, node.stableState.GetCurrentTerm()
	}
	node.stableState.SetVotedFor(args.candidateId)
	node.stableState.SetLastContact(time.Now())
	node.resetTimer()

	return true, node.stableState.GetCurrentTerm()
}
