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
	Term         int
	LeaderId     int32
	PrevLogIndex int
	PrevLogTerm  int
	Logs         []Log
	LeaderCommit int
}
type AppendEntriesResult struct {
	Term    int
	Success bool
}

func (raftRpc *RaftRPC) AppendEntries(args *AppendEntriesArgs, res *AppendEntriesResult) error {
	success, term := raftRpc.node.handleAppendEntries(args)
	res.Term = term
	res.Success = success
	return nil
}

type RequestVoteArgs struct {
	Term         int
	CandidateId  int32
	LastLogIndex int
	LastLogTerm  int
}
type RequestVoteResult struct {
	Term        int
	VoteGranted bool
}

func (raftRpc *RaftRPC) RequestVote(args *RequestVoteArgs, res *RequestVoteResult) error {
	success, term := raftRpc.node.handleRequestVote(args)
	res.Term = term
	res.VoteGranted = success
	return nil
}

func StartRpcServer(localAddress string, node *RaftNode) {
	raftRpc := &RaftRPC{node: node}
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

func (node *RaftNode) SendAppendEntries(peerId int32, args *AppendEntriesArgs) (*AppendEntriesResult, error) {
	clientAddress, ok := node.peers[peerId]
	if !ok {
		return nil, fmt.Errorf("no peer %d", peerId)
	}
	client, err := rpc.Dial("tcp", clientAddress)
	if err != nil {

		return nil, fmt.Errorf("no connection to peer %d", peerId)
	}
	defer client.Close()
	res := &AppendEntriesResult{}
	err = client.Call("RaftRPC.AppendEntries", args, res)
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %v", err)
	}
	return res, nil
}

func (node *RaftNode) SendRequestVote(peerId int32, args *RequestVoteArgs) (*RequestVoteResult, error) {
	clientAddress, ok := node.peers[peerId]
	if !ok {
		return nil, fmt.Errorf("no peer %d", peerId)
	}
	client, err := rpc.Dial("tcp", clientAddress)
	if err != nil {

		return nil, fmt.Errorf("no connection to peer %d", peerId)
	}
	defer client.Close()
	res := &RequestVoteResult{}
	err = client.Call("RaftRPC.RequestVote", args, res)
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %v", err)
	}
	return res, nil
}
