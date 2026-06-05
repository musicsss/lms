package presence

import (
	"sync"
	"time"
)

// Watcher represents a viewer on a video page, identified by user ID.
type Watcher struct {
	UserID    uint
	LastSeen  time.Time
}

// Hub tracks active watchers per video.
type Hub struct {
	mu       sync.Mutex
	watchers map[uint]map[uint]*Watcher // videoID -> userID -> Watcher
}

func NewHub() *Hub {
	return &Hub{
		watchers: make(map[uint]map[uint]*Watcher),
	}
}

// Heartbeat records that a user is currently watching a video.
func (h *Hub) Heartbeat(videoID, userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.watchers[videoID]; !ok {
		h.watchers[videoID] = make(map[uint]*Watcher)
	}
	h.watchers[videoID][userID] = &Watcher{
		UserID:   userID,
		LastSeen: time.Now(),
	}
}

// Count returns the number of watchers active within the given TTL.
func (h *Hub) Count(videoID uint, ttl time.Duration) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	users, ok := h.watchers[videoID]
	if !ok {
		return 0
	}

	cutoff := time.Now().Add(-ttl)
	var count int64
	for userID, w := range users {
		if w.LastSeen.After(cutoff) {
			count++
		} else {
			delete(users, userID)
		}
	}

	if len(users) == 0 {
		delete(h.watchers, videoID)
	}

	return count
}

// Cleanup removes stale entries older than ttl across all videos.
func (h *Hub) Cleanup(ttl time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	for vid, users := range h.watchers {
		for uid, w := range users {
			if w.LastSeen.Before(cutoff) {
				delete(users, uid)
			}
		}
		if len(users) == 0 {
			delete(h.watchers, vid)
		}
	}
}
