package raft

import (
	"net/rpc"
	"sync"
	"time"
)

type RaftNode struct {
	currentTerm int
	votedFor    int
	logStore    RaftLogStore

	commitIndex int
	lastApplied int

	nextIndex   int
	matchIndext int

	store Store

	lastContact     time.Time
	lastContactLock sync.RWMutex

	leaderAddr string
	leaderId   int

	state string
	peers map[int]*rpc.Client // key: id, string: client
}

func (node *RaftNode) run() {
	for {
		switch node.state {
		case "Follower":
			node.runFollower()
		case "Candidate":
			node.runCandidate()
		case "Leader":
			node.runLeader()
		}
	}
}

func (node *RaftNode) runFollower() {
	heartbeatTimer := randomTimeout()
	for node.state == "Follower" {
		select {
		case rpcCall := <-node:
			continue
		case <-heartbeatTimer:
			if time.Since(node.lastContact) < TIMEOUT {
				continue
			}
			// transition to candidate
			node.state = "Candidate"
		}
	}
}
func (node *RaftNode) runCandidate() {}
func (node *RaftNode) runLeader()    {}

func (node *RaftNode) handleAppendEntries(args *AppendEntriesArgs) (bool, int) {
	if args.term < node.currentTerm {
		return false, node.currentTerm
	}

	// handle 2
	// handle 3 and 4
	if args.leaderCommit > node.commitIndex {
		node.commitIndex = min(args.leaderCommit, len(node.logStore.logs))
	}
	return true, node.currentTerm
}

func (node *RaftNode) handleRequestVote(args *RequestVoteArgs) (bool, int) {
	if args.term < node.currentTerm {
		return false, node.currentTerm
	}

	// handle 2
	return true, node.currentTerm
}
