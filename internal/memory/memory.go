package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ansmeee-ai-agent/internal/config"
)

type entry struct {
	messages  []Message
	expiresAt time.Time
	createdAt time.Time
	agentID   string
}

// InMemoryStore is an in-memory session store with TTL support.
type InMemoryStore struct {
	mu    sync.RWMutex
	data  map[string]*entry
	ttl   time.Duration
	maxMs int
}

// NewInMemory creates a new in-memory store.
func NewInMemory(cfg *config.MemoryConfig) *InMemoryStore {
	s := &InMemoryStore{
		data:  make(map[string]*entry),
		ttl:   cfg.TTL,
		maxMs: cfg.MaxMessages,
	}
	go s.cleanup()
	return s
}

func (s *InMemoryStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, e := range s.data {
			if now.After(e.expiresAt) {
				delete(s.data, id)
			}
		}
		s.mu.Unlock()
	}
}

// AddMessage appends a message to a session, respecting the max message limit.
func (s *InMemoryStore) AddMessage(ctx context.Context, sessionID string, msg Message, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	e, ok := s.data[sessionID]
	if !ok {
		e = &entry{
			messages:  make([]Message, 0, s.maxMs),
			expiresAt: now.Add(s.ttl),
			createdAt: now,
		}
		s.data[sessionID] = e
	} else {
		e.expiresAt = time.Now().Add(s.ttl)
	}

	e.messages = append(e.messages, msg)
	if len(e.messages) > s.maxMs {
		e.messages = e.messages[len(e.messages)-s.maxMs:]
	}

	return nil
}

// History returns all messages for a session.
func (s *InMemoryStore) History(ctx context.Context, sessionID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	e.expiresAt = time.Now().Add(s.ttl)

	result := make([]Message, len(e.messages))
	copy(result, e.messages)
	return result, nil
}

// Exists checks whether a session exists.
func (s *InMemoryStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[sessionID]
	return ok, nil
}

// Delete removes a session.
func (s *InMemoryStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
	return nil
}

// ListSessions returns active sessions, optionally filtered by agentID.
func (s *InMemoryStore) ListSessions(ctx context.Context, userID int64, agentID string) ([]SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SessionInfo, 0, len(s.data))
	for id, e := range s.data {
		if agentID != "" && e.agentID != agentID {
			continue
		}
		title := ""
		if len(e.messages) > 0 {
			title = e.messages[0].Content
		}
		result = append(result, SessionInfo{
			ID:        id,
			Title:     title,
			AgentID:   e.agentID,
			CreatedAt: e.createdAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// SetAgent records which agent is used for a session.
func (s *InMemoryStore) SetAgent(ctx context.Context, sessionID, agentID string, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}
	e.agentID = agentID
	return nil
}

// GetAgent returns the agent for a session.
func (s *InMemoryStore) GetAgent(ctx context.Context, sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[sessionID]
	if !ok {
		return "", fmt.Errorf("session %q not found", sessionID)
	}
	return e.agentID, nil
}

// Close is a no-op for in-memory store.
func (s *InMemoryStore) Close() error { return nil }
