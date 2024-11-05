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
			// fmt.Println("TIMEOUT")
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
			fmt.Println("SENDING")
			response, err := node.SendRequestVote(peerId, request)
			fmt.Println(response)
			if err != nil {
				fmt.Println("Error")
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
				fmt.Printf("Node %d, now a follower", node.id)
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
			fmt.Println("Election Timeout. Restarting election")
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
		// fmt.Println("CURRENTLY HERE")
		// fmt.Println(command.Key)
		return node.store.Get(command.Key), nil
	}

	if node.state.Load() != LEADER {
		return "", fmt.Errorf("node is not leader")
	}
	fmt.Println(node.logStore.logs)
	node.logStore.AppendLog(command, node.stableState.GetCurrentTerm())
	quorum := 0
	updateChan := make(chan bool, len(node.peers)+1)
	fmt.Println(node.logStore.logs)
	for i := range node.peers {
		go node.UpdateFollower(i, node.followerState[i], updateChan)
	}
	updateChan <- true
	fmt.Println(quorum)
	for quorum < node.quorumSize() {
		<-updateChan
		quorum++
		fmt.Println(quorum)
	}
	fmt.Println(node.commitIndex)
	fmt.Println(node.logStore.GetLastLogIndex())
	fmt.Println(node.lastApplied)
	node.commitIndex = node.logStore.GetLastLogIndex()
	for node.lastApplied < node.commitIndex {
		fmt.Println(node.logStore.GetLogRange(node.lastApplied+1, node.commitIndex+1))
		node.store.ApplyLogs(node.logStore.GetLogRange(node.lastApplied+1, node.commitIndex+1))
		node.lastApplied = node.commitIndex
	}
	fmt.Println(node.store.dict)
	return "", nil
}

func (node *RaftNode) handleAppendEntries(args *AppendEntriesArgs) (bool, int) {
	fmt.Println("RECEIVING HEARTBEAT")
	node.stableState.SetLastContact(time.Now())
	node.resetTimer()
	if args.Term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}

	if args.Term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.Term)
	}

	// update leader
	node.leaderId.Store(args.LeaderId)

	if node.logStore.GetLastLogIndex() != 0 && (node.logStore.GetLastLogIndex() < args.PrevLogIndex || node.logStore.GetLastLogTerm() != args.Term) {
		return false, node.stableState.GetCurrentTerm()
	}
	// handle 3 and 4
	node.logStore.AppendEntries(args.Logs)
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
	fmt.Println("RECEIVING")
	fmt.Println(args.CandidateId)
	if time.Since(node.stableState.GetLastContact()) < minTimeout {
		return false, node.stableState.GetCurrentTerm()
	}
	fmt.Println("Condtion1 passed")

	if args.Term < node.stableState.GetCurrentTerm() {
		return false, node.stableState.GetCurrentTerm()
	}
	fmt.Println("Condtion 2 passed")

	if args.Term > node.stableState.GetCurrentTerm() {
		node.updateTerm(args.Term)
	}

	if node.stableState.GetVotedFor() != -1 && node.stableState.GetVotedFor() != args.CandidateId {
		return false, node.stableState.GetCurrentTerm()
	}
	fmt.Println("Condtion 3 passed")

	if node.logStore.GetLastLogTerm() > args.LastLogTerm ||
		node.logStore.GetLastLogIndex() > args.LastLogIndex {
		return false, node.stableState.GetCurrentTerm()
	}
	fmt.Println("Condtion 4 passed")
	node.stableState.SetVotedFor(args.CandidateId)
	node.stableState.SetLastContact(time.Now())
	node.resetTimer()

	return true, node.stableState.GetCurrentTerm()
}
