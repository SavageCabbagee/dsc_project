package raft

import (
	"fmt"
	"sync"
)

type Command struct {
	Operation string `json:"operation"` // SET, DELETE
	Key       string `json:"key,omitempty"`
	Value     string `json:"value,omitempty"`
}

type Store struct {
	dict map[string]string
	lock *sync.Mutex
}

func (s *Store) Get(key string) string {
	// fmt.Println(s.dict)
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.dict[key]
}

func (s *Store) Set(key, value string) {
	s.dict[key] = value
}

func (s *Store) Delete(key string) {
	fmt.Printf("Deleting key: %s\n", key)
	delete(s.dict, key)
}

func (s *Store) ApplyLogs(logs []Log) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, log := range logs {
		command := log.Command
		switch command.Operation {
		case "SET":
			s.Set(command.Key, command.Value)
		case "DELETE":
			s.Delete(command.Key)
		}
	}
}
