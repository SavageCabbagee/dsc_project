package raft

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type RaftHTTPServer struct {
	addr string
	ln   net.Listener

	node *RaftNode
}

type KeyValueRequest struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type NodeStatus struct {
	NodeId       int32 `json:"nodeId"`
	State        int32 `json:"state"`
	CurrentTerm  int   `json:"currentTerm"`
	VotedFor     int32 `json:"votedFor"`
	LeaderId     int32 `json:"leaderId"`
	LastLogIndex int   `json:"lastLogIndex"`
	LastLogTerm  int   `json:"lastLogTerm"`
	Logs         []Log `json:"logs"`
}

func NewHttpServer(addr string, node *RaftNode) *RaftHTTPServer {
	return &RaftHTTPServer{
		addr: addr,
		node: node,
	}
}

func (s *RaftHTTPServer) Start() error {
	server := http.Server{
		Handler: s,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln

	http.Handle("/", s)

	go func() {
		err := server.Serve(s.ln)
		if err != nil {
			log.Fatalf("HTTP serve: %s", err)
		}
	}()

	return nil
}

func (server *RaftHTTPServer) Close() {
	server.ln.Close()
}

func (server *RaftHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/key") {
		server.handleKeyRequest(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/status") {
		server.handleStatus(w, r)
	} else if strings.HasPrefix(r.URL.Path, "/kill") {
		server.handleKill(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (server *RaftHTTPServer) handleKeyRequest(w http.ResponseWriter, r *http.Request) {
	getKey := func() string {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 3 {
			return ""
		}
		return parts[2]
	}
	key := getKey()

	switch r.Method {
	case "GET":
		// fmt.Println("HERE")
		// fmt.Println(key)
		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
		}

		// fmt.Println(key)
		value, err := server.node.handleCommand(Command{"GET", key, ""})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// fmt.Println(value)

		response := KeyValueRequest{
			Key:   key,
			Value: value,
		}
		json.NewEncoder(w).Encode(response)

	case "POST":
		m := map[string]string{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for key, value := range m {
			if _, err := server.node.handleCommand(Command{"SET", key, value}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

	case "DELETE":
		var request KeyValueRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := server.node.handleCommand(Command{"DELETE", key, request.Value}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (server *RaftHTTPServer) handleStatus(w http.ResponseWriter, _ *http.Request) {
	status := NodeStatus{
		NodeId:       server.node.id,
		State:        server.node.state.Load(),
		CurrentTerm:  server.node.stableState.GetCurrentTerm(),
		VotedFor:     server.node.stableState.GetVotedFor(),
		LeaderId:     server.node.leaderId.Load(),
		LastLogIndex: server.node.logStore.GetLastLogIndex(),
		LastLogTerm:  server.node.logStore.GetLastLogTerm(),
		Logs:         server.node.logStore.GetLogsFrom(1),
	}

	json.NewEncoder(w).Encode(status)
}

func (server *RaftHTTPServer) handleKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		fmt.Printf("Node %d initiating graceful shutdown...\n", server.node.id)

		// Step down if leader
		// if server.node.state.Load() == LEADER {
		// 	server.node.state.Store(FOLLOWER)
		// 	server.node.leaderId.Store(-1)
		// }

		// Wait a moment for state changes to propagate
		time.Sleep(100 * time.Millisecond)
		server.Close()
		fmt.Printf("Node %d shutdown complete\n", server.node.id)
		os.Exit(0)
	}()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node %d shutdown initiated\n", server.node.id)
}
