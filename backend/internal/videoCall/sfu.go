package main

import (
	"log"
	"sync"
)

// SFU is the top-level Selective Forwarding Unit.
// It owns the room registry and is the single entry point for the HTTP layer.
type SFU struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewSFU() *SFU {
	return &SFU{
		rooms: make(map[string]*Room),
	}
}

// GetOrCreateRoom returns an existing room or creates a new one atomically.
func (s *SFU) GetOrCreateRoom(id string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.rooms[id]; ok {
		return r
	}

	r := newRoom(id, func() {
		// Called by Room when it becomes empty — clean up our registry.
		s.mu.Lock()
		delete(s.rooms, id)
		s.mu.Unlock()
		log.Printf("[sfu] room %s removed (empty)", id)
	})

	s.rooms[id] = r
	log.Printf("[sfu] room %s created", id)
	return r
}
