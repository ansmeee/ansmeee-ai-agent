package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ansmeee-ai-agent/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements SessionStore backed by Redis.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
	maxMs  int
	prefix string
}

// NewRedis creates a new Redis-backed session store.
func NewRedis(cfg *config.RedisConfig, memCfg *config.MemoryConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:       cfg.Addr,
		Password:   cfg.Password,
		DB:         cfg.DB,
		MaxRetries: cfg.MaxRetries,
		PoolSize:   cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &RedisStore{
		client: client,
		ttl:    memCfg.TTL,
		maxMs:  memCfg.MaxMessages,
		prefix: "session:",
	}, nil
}

func (r *RedisStore) key(sessionID string) string {
	return r.prefix + sessionID
}

// AddMessage appends a message to a session in Redis.
func (r *RedisStore) AddMessage(ctx context.Context, sessionID string, msg Message, userID int64) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	pipe := r.client.Pipeline()
	pipe.RPush(ctx, r.key(sessionID), data)
	pipe.Expire(ctx, r.key(sessionID), r.ttl)
	pipe.LTrim(ctx, r.key(sessionID), -int64(r.maxMs), -1)

	_, err = pipe.Exec(ctx)
	return err
}

// History returns all messages for a session from Redis.
func (r *RedisStore) History(ctx context.Context, sessionID string) ([]Message, error) {
	vals, err := r.client.LRange(ctx, r.key(sessionID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lrange: %w", err)
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}

	// Refresh TTL.
	r.client.Expire(ctx, r.key(sessionID), r.ttl)

	messages := make([]Message, 0, len(vals))
	for _, v := range vals {
		var msg Message
		if err := json.Unmarshal([]byte(v), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal message: %w", err)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// Exists checks whether a session key exists in Redis.
func (r *RedisStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	n, err := r.client.Exists(ctx, r.key(sessionID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Delete removes a session from Redis.
func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	return r.client.Del(ctx, r.key(sessionID)).Err()
}

// ListSessions scans Redis for session keys, optionally filtered by agentID.
func (r *RedisStore) ListSessions(ctx context.Context, userID int64, agentID string) ([]SessionInfo, error) {
	var result []SessionInfo
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, r.prefix+"*", 50).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}
		for _, key := range keys {
			id := key[len(r.prefix):]
			storedAgent, _ := r.client.Get(ctx, key+":agent").Result()
			if agentID != "" && storedAgent != agentID {
				continue
			}
			vals, err := r.client.LRange(ctx, key, 0, 0).Result()
			title := ""
			if err == nil && len(vals) > 0 {
				var msg Message
				if json.Unmarshal([]byte(vals[0]), &msg) == nil {
					title = msg.Content
				}
			}
			result = append(result, SessionInfo{
				ID:      id,
				Title:   title,
				AgentID: storedAgent,
			})
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// SetAgent records which agent is used for a session.
func (r *RedisStore) SetAgent(ctx context.Context, sessionID, agentID string, userID int64) error {
	return r.client.Set(ctx, r.prefix+sessionID+":agent", agentID, r.ttl).Err()
}

// GetAgent returns the agent for a session.
func (r *RedisStore) GetAgent(ctx context.Context, sessionID string) (string, error) {
	return r.client.Get(ctx, r.prefix+sessionID+":agent").Result()
}

// Close releases the Redis connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}
