package mcpproxy

import "sync"

// SessionManager defines the interface for managing session information
type SessionManager interface {
	// Set stores a sessionId and its corresponding backend endpoint
	Set(sessionId, endpoint string)
	// Get retrieves the backend endpoint for a sessionId
	Get(sessionId string) (string, bool)
	// Delete removes a sessionId from the store
	Delete(sessionId string)
}

// MemStore implements the SessionManager interface
type MemStore struct {
	mu       sync.RWMutex
	sessions map[string]string // sessionId -> host+endpoint
}

// NewMemStore creates a new session store
func NewMemStore() *MemStore {
	return &MemStore{
		sessions: make(map[string]string),
	}
}

// Set stores a sessionId and its corresponding backend endpoint
func (s *MemStore) Set(sessionId, endpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionId] = endpoint
}

// Get retrieves the backend endpoint for a sessionId
func (s *MemStore) Get(sessionId string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	endpoint, ok := s.sessions[sessionId]
	return endpoint, ok
}

// Delete removes a sessionId from the store
func (s *MemStore) Delete(sessionId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionId)
}
