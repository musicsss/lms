package loginprotect

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/lms/server/internal/runtimecfg"
)

// 登录保护策略中的适用范围类型
const (
	RangeAllUser    = "ALL_USER"    // 全局策略
	RangeSingleUser = "SINGLE_USER" // 指定用户策略
	RangeIP         = "IP"          // 指定 IP 策略
)

// 封禁粒度
const (
	BlockByAccount = "ACCOUNT" // 按账户封禁
	BlockByIP      = "IP"      // 按 IP 封禁
)

// Guard.Check 返回值
const (
	ActionOK      = "ok"      // 允许登录
	ActionBlocked = "blocked" // 已封禁
	ActionCaptcha = "captcha" // 需要验证码
)

// 时间与阈值常量
const (
	windowDuration    = 5 * time.Minute // 失败计数窗口
	blockDuration     = 1 * time.Hour   // 封禁时长
	captchaThreshold  = 3              // 触发验证码的失败次数
	blockThreshold    = 5              // 触发封禁的失败次数
	captchaTTL        = 5 * time.Minute // 验证码有效期
	cleanupInterval   = 2 * time.Minute // 过期数据清理间隔
	captchaMaxOperand = 20             // 验证码运算数最大值 (1-20)
)

// 去重键分隔符
const dedupSep = "|"

// 默认策略：全局按 IP 封禁
var defaultPolicy = Policy{RangeType: RangeAllUser, BlockBy: BlockByIP}

// Policy 表示一条登录失败封禁策略。
type Policy struct {
	RangeType string // ALL_USER / SINGLE_USER / IP
	RangeVal  string // 用户名 (SINGLE_USER) 或 IP 地址 (IP)
	BlockBy   string // ACCOUNT 或 IP
}

// Guard 跟踪登录尝试，提供限流、验证码和封禁逻辑。
type Guard struct {
	mu              sync.Mutex
	attempts        map[string]*attemptWindow // key = IP
	accountAttempts map[string]*attemptWindow // key = username
	captchas        map[string]*captchaItem
	policies        []Policy
}

type attemptWindow struct {
	failures     []time.Time
	blockedUntil time.Time
}

type captchaItem struct {
	answer    string
	expiresAt time.Time
}

// NewGuard 创建 Guard 实例并启动后台清理协程。
func NewGuard() *Guard {
	g := &Guard{
		attempts:        make(map[string]*attemptWindow),
		accountAttempts: make(map[string]*attemptWindow),
		captchas:        make(map[string]*captchaItem),
		policies:        []Policy{defaultPolicy},
	}
	go g.cleanupLoop()
	return g
}

// ApplyPolicies 从运行时配置加载封禁策略，按 (RangeType, RangeVal, BlockBy) 去重。
// 后加载的策略覆盖先加载的。
func (g *Guard) ApplyPolicies(rows []runtimecfg.RuntimeConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()

	policies := make([]Policy, 0)
	seen := make(map[string]bool)
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		p := Policy{
			RangeType: attrs[runtimecfg.FieldRange],
			BlockBy:   attrs[runtimecfg.FieldBlockPolicy],
		}
		// 解析 SINGLE_USER:username 或 IP:ipaddr 格式
		if idx := strings.Index(p.RangeType, ":"); idx >= 0 {
			p.RangeVal = p.RangeType[idx+1:]
			p.RangeType = p.RangeType[:idx]
		}
		// 按 (RangeType, RangeVal, BlockBy) 去重
		key := p.RangeType + dedupSep + p.RangeVal + dedupSep + p.BlockBy
		if seen[key] {
			for i, existing := range policies {
				ek := existing.RangeType + dedupSep + existing.RangeVal + dedupSep + existing.BlockBy
				if ek == key {
					policies[i] = p
					break
				}
			}
			continue
		}
		seen[key] = true
		policies = append(policies, p)
	}
	if len(policies) == 0 {
		policies = []Policy{defaultPolicy}
	}
	g.policies = policies
}

// ClearAll 清除所有限流计数器和封禁状态。
func (g *Guard) ClearAll() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.attempts = make(map[string]*attemptWindow)
	g.accountAttempts = make(map[string]*attemptWindow)
}

// resolveBlockBy 根据当前策略集决定应使用的封禁粒度。
// 策略按优先级从高到低匹配：SINGLE_USER > IP > ALL_USER。
func (g *Guard) resolveBlockBy(ip, username string) string {
	blockBy := BlockByIP
	for _, p := range g.policies {
		switch p.RangeType {
		case RangeSingleUser:
			if username != "" && (p.RangeVal == "" || p.RangeVal == username) {
				blockBy = p.BlockBy
			}
		case RangeIP:
			if p.RangeVal == "" || p.RangeVal == ip {
				blockBy = p.BlockBy
			}
		case RangeAllUser:
			blockBy = p.BlockBy
		}
	}
	return blockBy
}

// Check 返回登录前需要执行的动作：ok / captcha / blocked。
func (g *Guard) Check(ip, username string) (string, time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	blockBy := g.resolveBlockBy(ip, username)

	// 选择对应的尝试记录窗口
	var w *attemptWindow
	if blockBy == BlockByAccount && username != "" {
		w = g.accountAttempts[username]
	} else {
		w = g.attempts[ip]
	}

	if w == nil {
		return ActionOK, time.Time{}
	}

	// 检查是否处于封禁期
	if now.Before(w.blockedUntil) {
		return ActionBlocked, w.blockedUntil
	}

	// 清理过期失败记录
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
		return ActionBlocked, w.blockedUntil
	}
	if count >= captchaThreshold {
		return ActionCaptcha, time.Time{}
	}
	return ActionOK, time.Time{}
}

// RecordFailure 记录一次登录失败。
func (g *Guard) RecordFailure(ip, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	blockBy := g.resolveBlockBy(ip, username)
	now := time.Now()

	// 按账户封禁时，额外记录账户维度的失败
	if blockBy == BlockByAccount && username != "" {
		if g.accountAttempts[username] == nil {
			g.accountAttempts[username] = &attemptWindow{}
		}
		g.accountAttempts[username].failures = append(g.accountAttempts[username].failures, now)
	}
	// 始终记录 IP 维度的失败
	if g.attempts[ip] == nil {
		g.attempts[ip] = &attemptWindow{}
	}
	g.attempts[ip].failures = append(g.attempts[ip].failures, now)
}

// RecordSuccess 登录成功后清除该 IP 和用户的所有失败记录。
func (g *Guard) RecordSuccess(ip, username string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attempts, ip)
	if username != "" {
		delete(g.accountAttempts, username)
	}
}

// GenerateCaptcha 生成算术验证码 (加法)，返回 id、题目和答案。
func (g *Guard) GenerateCaptcha() (id, question, answer string) {
	a, _ := rand.Int(rand.Reader, big.NewInt(captchaMaxOperand))
	b, _ := rand.Int(rand.Reader, big.NewInt(captchaMaxOperand))
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

// VerifyCaptcha 校验验证码答案是否正确且未过期。
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

// cleanupLoop 定期清理过期的验证码和失败记录。
func (g *Guard) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		g.cleanup()
	}
}

// cleanup 删除过期的验证码，并清理已解封且无有效失败记录的窗口。
func (g *Guard) cleanup() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// 清理过期验证码
	for id, item := range g.captchas {
		if now.After(item.expiresAt) {
			delete(g.captchas, id)
		}
	}

	cutoff := now.Add(-windowDuration)
	// 清理 IP 维度的记录
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
	// 清理账户维度的记录
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
