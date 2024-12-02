package raft

import "fmt"

func (node *RaftNode) SendHeartbeat(peerId int32, followerState *FollowerState) {
	req := &AppendEntriesArgs{node.stableState.GetCurrentTerm(), node.id, followerState.nextIndex - 1, node.logStore.GetLog(followerState.nextIndex - 1).Term, node.logStore.logs, node.commitIndex}
	// do not care about the reply
	node.SendAppendEntries(peerId, req)
}

type FollowerState struct {
	nextIndex  int
	matchIndex int
}

func (node *RaftNode) UpdateFollower(peerId int32, followerState *FollowerState, updateChan chan bool) {
	success := false
	for !success {
		req := &AppendEntriesArgs{node.stableState.GetCurrentTerm(), node.id, followerState.nextIndex - 1, node.logStore.GetLog(followerState.nextIndex - 1).Term, node.logStore.GetLogsFrom(followerState.nextIndex), node.commitIndex}
		res, err := node.SendAppendEntries(peerId, req)
		if err != nil {
			fmt.Printf("ERROR %v", err)
			return
		}
		if res.Term > node.stableState.GetCurrentTerm() {
			node.updateTerm(res.Term)
			return
		}
		if !res.Success {
			followerState.nextIndex--
			continue
		}
		success = true
	}
	followerState.nextIndex = node.logStore.GetLastLogIndex() + 1
	followerState.matchIndex = node.logStore.GetLastLogIndex()
	updateChan <- true
}
