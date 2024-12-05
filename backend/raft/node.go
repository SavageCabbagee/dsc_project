package raft

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	FOLLOWER  int32 = 0
	CANDIDATE int32 = 1
	LEADER    int32 = 2
)

type RaftNode struct {
	id          int32
	stableState *StableState
	logStore    *RaftLogStore

	commitIndex int
	lastApplied int

	store *Store

	heartbeatTimer <-chan time.Time

	leaderId atomic.Int32

	state atomic.Int32
	peers map[int32]string // key: id, string: client

	// leader state
	followerState map[int32]*FollowerState
}

func (node *RaftNode) quorumSize() int {
	return (len(node.peers)+1)/2 + 1
}

func (node *RaftNode) updateTerm(term int) {
	node.stableState.SetCurrentTerm(term)
	node.stableState.SetVotedFor(-1)
	node.state.Store(FOLLOWER)
}

func (node *RaftNode) resetTimer() {
	node.heartbeatTimer = randomTimeout()
}

func NewRaftNode(id int32, peers map[int32]string) *RaftNode {
	state := &StableState{0, -1, time.Now(), &sync.RWMutex{}}
	logStore := &RaftLogStore{make([]Log, 0), &sync.RWMutex{}}
	store := &Store{make(map[string]string), &sync.Mutex{}}
	node := &RaftNode{id, state, logStore, 0, 0, store, make(<-chan time.Time), atomic.Int32{}, atomic.Int32{}, peers, map[int32]*FollowerState{}}
	node.leaderId.Store(-1)
	return node
}

func (node *RaftNode) Run() {
	for {
		switch node.state.Load() {
		case FOLLOWER:
			fmt.Println("FOLLOWER")
			node.runFollower()
		case CANDIDATE:
			fmt.Println("CANDIDATE")
			node.runCandidate()
		case LEADER:
			fmt.Println("LEADER")
			node.runLeader()
		}
	}
}

func (node *RaftNode) runFollower() {
	node.resetTimer()
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
	responseChannel := make(chan *RequestVoteResult, len(node.peers)+1)
	request := &RequestVoteArgs{node.stableState.GetCurrentTerm(), node.id, node.logStore.GetLastLogIndex(), node.logStore.GetLastLogTerm()}
	for id := range node.peers {
		go func(peerId int32) {
			response, err := node.SendRequestVote(peerId, request)
			if err != nil {
				fmt.Println(err)
				return
			}
			responseChannel <- response
		}(id)
	}
	responseChannel <- &RequestVoteResult{node.stableState.GetCurrentTerm(), true}

	votesNeeded := node.quorumSize()
	grantedVotes := 0
	for node.state.Load() == CANDIDATE {
		select {
		case res := <-responseChannel:
			if res.Term > node.stableState.GetCurrentTerm() {
				node.state.Store(FOLLOWER)
				node.stableState.SetCurrentTerm(res.Term)
				node.stableState.SetLastContact(time.Now())
				return
			}
			if res.VoteGranted {
				grantedVotes++
			}

			if grantedVotes >= votesNeeded {
				node.state.Store(LEADER)
				node.leaderId.Store(node.id)
				return
			}
		case <-node.heartbeatTimer:
			return
		}
	}

}

func (node *RaftNode) runLeader() {
	node.followerState = make(map[int32]*FollowerState)
	for i := range node.peers {
		node.followerState[i] = &FollowerState{node.logStore.GetLastLogIndex() + 1, 0}
	}
	leaderHeartbeat := leaderHeartbeatTimeout()
	for node.state.Load() == LEADER {
		<-leaderHeartbeat
		for i := range node.peers {
			go node.SendHeartbeat(i, node.followerState[i])
		}
		leaderHeartbeat = leaderHeartbeatTimeout()
	}
}

func (node *RaftNode) handleCommand(command Command) (string, error) {
	if command.Operation == "GET" {
		return node.store.Get(command.Key), nil
	}
	if node.state.Load() != LEADER {
		if command.Operation == "SET" || command.Operation == "DELETE" {
			// If not leader and is to set, then send request to leader.
			strval, errorval := node.RelayCommandFromNode(node.leaderId.Load(), command)

			return strval, errorval
		}
		return "", fmt.Errorf("node is not leader")
	}

	node.logStore.AppendLog(command, node.stableState.GetCurrentTerm())
	quorum := 0
	updateChan := make(chan bool, len(node.peers)+1)

	for i := range node.peers {
		go node.UpdateFollower(i, node.followerState[i], updateChan)
	}
	updateChan <- true
	for quorum < node.quorumSize() {
		<-updateChan
		quorum++
	}
	node.commitIndex = node.logStore.GetLastLogIndex()
	for node.lastApplied < node.commitIndex {
		node.store.ApplyLogs(node.logStore.GetLogRange(node.lastApplied+1, node.commitIndex+1))
		node.lastApplied = node.commitIndex
	}
	return "", nil
}

func (node *RaftNode) handleAppendEntries(args *AppendEntriesArgs) (bool, int) {

	node.stableState.SetLastContact(time.Now())
	node.resetTimer()
	if args.Term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.Term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.Term)
	}

	node.leaderId.Store(args.LeaderId)
	if node.logStore.GetLastLogIndex() != 0 && (node.logStore.GetLastLogIndex() < args.PrevLogIndex || node.logStore.GetLastLogTerm() != args.PrevLogTerm) {
		return false, node.stableState.GetCurrentTerm()
	}
	node.logStore.AppendEntries(args.Logs)
	fmt.Println("LOG: ", node.logStore.logs)
	if args.LeaderCommit > node.commitIndex {
		node.commitIndex = min(args.LeaderCommit, node.logStore.GetLastLogIndex())
		for node.lastApplied < node.commitIndex {
			node.store.ApplyLogs(node.logStore.GetLogRange(node.lastApplied+1, node.commitIndex+1))
			node.lastApplied = node.commitIndex
		}
	}
	return true, node.stableState.GetCurrentTerm()
}

func (node *RaftNode) handleRequestVote(args *RequestVoteArgs) (bool, int) {
	if time.Since(node.stableState.GetLastContact()) < minTimeout {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.Term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.Term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.Term)
	}

	if node.stableState.GetVotedFor() != -1 && node.stableState.GetVotedFor() != args.CandidateId {
		return false, node.stableState.GetCurrentTerm()
	}

	if node.logStore.GetLastLogTerm() > args.LastLogTerm ||
		node.logStore.GetLastLogIndex() > args.LastLogIndex {
		return false, node.stableState.GetCurrentTerm()
	}
	node.stableState.SetVotedFor(args.CandidateId)
	node.stableState.SetLastContact(time.Now())
	node.resetTimer()

	return true, node.stableState.GetCurrentTerm()
}
