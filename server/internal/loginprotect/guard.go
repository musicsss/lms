package loginprotect

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/lms/server/internal/runtimecfg"
)

// Policy represents a login failure blocking policy from runtime config.
type Policy struct {
	RangeType string // ALL_USER, SINGLE_USER, IP
	RangeVal  string // username (for SINGLE_USER) or ip (for IP)
	BlockBy   string // ACCOUNT or IP
}

// Guard tracks login attempts and provides rate-limiting + captcha logic.
type Guard struct {
	mu         sync.Mutex
	attempts   map[string]*attemptWindow
	accountAttempts map[string]*attemptWindow  // keyed by username
	captchas   map[string]*captchaItem
	policies   []Policy
}

type attemptWindow struct {
	failures     []time.Time
	blockedUntil time.Time
}

type captchaItem struct {
	answer    string
	expiresAt time.Time
}

func NewGuard() *Guard {
	g := &Guard{
		attempts:        make(map[string]*attemptWindow),
		accountAttempts: make(map[string]*attemptWindow),
		captchas:        make(map[string]*captchaItem),
		policies:        []Policy{{RangeType: "ALL_USER", BlockBy: "IP"}},
	}
	go g.cleanupLoop()
	return g
}

const (
	windowDuration   = 5 * time.Minute
	blockDuration    = 1 * time.Hour
	captchaThreshold = 3
	blockThreshold   = 5
	captchaTTL       = 5 * time.Minute
)

// ApplyPolicies updates the active blocking policies from runtime config.
func (g *Guard) ApplyPolicies(rows []runtimecfg.RuntimeConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()

	policies := make([]Policy, 0)
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		p := Policy{
			RangeType: attrs["RANGE"],
			BlockBy:   attrs["BLOCKPLCY"],
		}
		// Parse SINGLE_USER:username or IP:ipaddr format
		if idx := stringsIdx(p.RangeType, ":"); idx >= 0 {
			p.RangeVal = p.RangeType[idx+1:]
			p.RangeType = p.RangeType[:idx]
		}
		policies = append(policies, p)
	}
	if len(policies) == 0 {
		policies = []Policy{{RangeType: "ALL_USER", BlockBy: "IP"}}
	}
	g.policies = policies
}

// ClearAll resets all rate limit counters.
func (g *Guard) ClearAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.attempts = make(map[string]*attemptWindow)
	g.accountAttempts = make(map[string]*attemptWindow)
}

// Check returns the action required before login can proceed.
func (g *Guard) Check(ip, username string) (string, time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// find matching policy
	blockBy := "IP"
	for _, p := range g.policies {
		switch p.RangeType {
		case "SINGLE_USER":
			if username != "" && (p.RangeVal == "" || p.RangeVal == username) {
				blockBy = p.BlockBy
			}
		case "IP":
			if p.RangeVal == "" || p.RangeVal == ip {
				blockBy = p.BlockBy
			}
		case "ALL_USER":
			blockBy = p.BlockBy
		}
	}

	// select the right window based on blockBy
	var w *attemptWindow
	if blockBy == "ACCOUNT" && username != "" {
		w = g.accountAttempts[username]
	} else {
		w = g.attempts[ip]
	}

	if w == nil {
		return "ok", time.Time{}
	}

	if now.Before(w.blockedUntil) {
		return "blocked", w.blockedUntil
	}

	cutoff := now.Add(-windowDuration)
	valid := make([]time.Time, 0, len(w.failures))
	for _, t := range w.failures {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	w.failures = valid

	count := len(valid)
	if count >= blockThreshold {
		w.blockedUntil = now.Add(blockDuration)
		return "blocked", w.blockedUntil
	}
	if count >= captchaThreshold {
		return "captcha", time.Time{}
	}
	return "ok", time.Time{}
}

// RecordFailure records a failed login attempt.
func (g *Guard) RecordFailure(ip, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// determine which window(s) to update
	blockBy := "IP"
	for _, p := range g.policies {
		switch p.RangeType {
		case "SINGLE_USER":
			if username != "" && (p.RangeVal == "" || p.RangeVal == username) {
				blockBy = p.BlockBy
			}
		case "IP":
			if p.RangeVal == "" || p.RangeVal == ip {
				blockBy = p.BlockBy
			}
		case "ALL_USER":
			blockBy = p.BlockBy
		}
	}

	now := time.Now()
	if blockBy == "ACCOUNT" && username != "" {
		if g.accountAttempts[username] == nil {
			g.accountAttempts[username] = &attemptWindow{}
		}
		g.accountAttempts[username].failures = append(g.accountAttempts[username].failures, now)
	}
	if g.attempts[ip] == nil {
		g.attempts[ip] = &attemptWindow{}
	}
	g.attempts[ip].failures = append(g.attempts[ip].failures, now)
}

// RecordSuccess clears attempts on successful login.
func (g *Guard) RecordSuccess(ip, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attempts, ip)
	if username != "" {
		delete(g.accountAttempts, username)
	}
}

// GenerateCaptcha creates a math captcha.
func (g *Guard) GenerateCaptcha() (id, question, answer string) {
	a, _ := rand.Int(rand.Reader, big.NewInt(20))
	b, _ := rand.Int(rand.Reader, big.NewInt(20))
	aVal := int(a.Int64()) + 1
	bVal := int(b.Int64()) + 1

	id = fmt.Sprintf("%x", time.Now().UnixNano())
	question = fmt.Sprintf("%d + %d = ?", aVal, bVal)
	answer = fmt.Sprintf("%d", aVal+bVal)

	g.mu.Lock()
	g.captchas[id] = &captchaItem{answer: answer, expiresAt: time.Now().Add(captchaTTL)}
	g.mu.Unlock()

	return id, question, answer
}

// VerifyCaptcha checks a captcha answer.
func (g *Guard) VerifyCaptcha(id, answer string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	item, ok := g.captchas[id]
	if !ok {
		return false
	}
	if time.Now().After(item.expiresAt) {
		delete(g.captchas, id)
		return false
	}
	if item.answer != answer {
		return false
	}
	delete(g.captchas, id)
	return true
}

func stringsIdx(s, sep string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			return i
		}
	}
	return -1
}

func (g *Guard) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		g.cleanup()
	}
}

func (g *Guard) cleanup() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	for id, item := range g.captchas {
		if now.After(item.expiresAt) {
			delete(g.captchas, id)
		}
	}

	cutoff := now.Add(-windowDuration)
	for ip, w := range g.attempts {
		if now.After(w.blockedUntil) {
			valid := make([]time.Time, 0)
			for _, t := range w.failures {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(g.attempts, ip)
			} else {
				w.failures = valid
			}
		}
	}
	for user, w := range g.accountAttempts {
		if now.After(w.blockedUntil) {
			valid := make([]time.Time, 0)
			for _, t := range w.failures {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(g.accountAttempts, user)
			} else {
				w.failures = valid
			}
		}
	}
}
