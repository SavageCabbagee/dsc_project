package raft

import (
	"sync"
)

type Command struct {
	Operation string `json:"operation"` // SET, DELETE
	Key       string `json:"key,omitempty"`
	Value     string `json:"value,omitempty"`
}

type Store struct {
	mu   sync.Mutex
	dict map[string]string
}

func (s *Store) Get(key string) string {
	return s.dict[key]
}

func (s *Store) Set(key, value string) {
	s.dict[key] = value
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dict, key)
}

func (s *Store) ApplyLogs(logs []Log) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
