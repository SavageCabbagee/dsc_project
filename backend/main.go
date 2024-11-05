package main

import (
	"backend/raft"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultHTTPAddr = "localhost:11000"
	DefaultRaftAddr = "localhost:12000"
)

var httpAddr string
var raftAddr string
var nodeID string
var peers string

func init() {
	flag.StringVar(&httpAddr, "haddr", DefaultHTTPAddr, "Set the HTTP bind address")
	flag.StringVar(&raftAddr, "raddr", DefaultRaftAddr, "Set Raft bind address")
	flag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	flag.StringVar(&peers, "peers", "", "comma-separated key=value pairs (e.g., 'key1=value1,key2=value2')")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	peersMapping := make(map[int32]string)
	if peers != "" {
		// Split by comma
		for _, pair := range strings.Split(peers, ",") {
			// Split by =
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				id, _ := strconv.Atoi(kv[0])
				peersMapping[int32(id)] = kv[1]
			}
		}
	}
	nodeId, _ := strconv.Atoi(nodeID)
	node := raft.NewRaftNode(int32(nodeId), peersMapping)
	server := raft.NewHttpServer(httpAddr, node)
	server.Start()
	raft.StartRpcServer(raftAddr, node)
	node.Run()
}
