package raft

import (
	"fmt"
	"net"
	"net/rpc"
)

type RaftRPC struct {
	node *RaftNode
}

type AppendEntriesArgs struct {
	term         int
	leaderId     int
	prevLogIndex int
	prevLogTerm  int
	logEntries   []Log
	leaderCommit int
}
type AppendEntriesResult struct {
	term    int
	success bool
}

func (raftRpc *RaftRPC) AppendEntries(args *AppendEntriesArgs, res *AppendEntriesResult) {
	success, term := raftRpc.node.handleAppendEntries(args)
	res.term = term
	res.success = success
}

type RequestVoteArgs struct {
	term         int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}
type RequestVoteResult struct {
	term        int
	voteGranted bool
}

func (raftRpc *RaftRPC) RequestVote(args *RequestVoteArgs, res *RequestVoteResult) {
	success, term := raftRpc.node.handleRequestVote(args)
	res.term = term
	res.voteGranted = success

}

func StartRpcServer(localAddress string) {
	raftRpc := new(RaftRPC)
	rpcServer := rpc.NewServer()
	err := rpcServer.Register(raftRpc)
	if err != nil {
		fmt.Errorf("failed to register RPC: %v", err)
	}

	listener, err := net.Listen("tcp", localAddress)
	if err != nil {
		fmt.Errorf("failed to start RPC listener: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Errorf("RPC accept error: %v\n", err)
				continue
			}
			go rpcServer.ServeConn(conn)
		}
	}()
}

func (node *RaftNode) sendAppendEntries(peerId int, args *AppendEntriesArgs) *AppendEntriesResult {
	client, err := node.peers[peerId]
	if !err {
		fmt.Errorf("no connection to peer %d", peerId)
		return nil
	}
	res := &AppendEntriesResult{}
	err1 := client.Call("Raft.AppendEntries", args, res)
	if err1 != nil {
		fmt.Errorf("RPC call failed: %v", err)
		return nil
	}
	return res
}

func (node *RaftNode) sendRequestVote(peerId int, args *RequestVoteArgs) *RequestVoteResult {
	client, err := node.peers[peerId]
	if !err {
		fmt.Errorf("no connection to peer %d", peerId)
		return nil
	}
	res := &RequestVoteResult{}
	err1 := client.Call("Raft.RequestVote", args, res)
	if err1 != nil {
		fmt.Errorf("RPC call failed: %v", err)
		return nil
	}
	return res
}
