package runtimecfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// NotifyFunc is called after a config change with the changed targets.
type NotifyFunc func(target string)

// Engine is the central runtime config engine.
type Engine struct {
	store     *Store
	cache     *Cache
	listeners []NotifyFunc
}

func NewEngine(store *Store) *Engine {
	return &Engine{store: store, cache: NewCache()}
}

// Start loads all configs from DB, ensures SET defaults exist, and initializes the cache.
func (e *Engine) Start() error {
	rows, err := e.store.LoadAll()
	if err != nil {
		return err
	}

	// ensure defaults for SET configs
	defaults := map[string]map[string]string{
		"SYSLOG":   {"LEVEL": "INFO"},
		"JWT":      {"EXPIRETIME": "72"},
		"FILEUPLD": {"MAXSIZE": "2048"},
	}
	for target, attrs := range defaults {
		if !e.store.HasSet(target) {
			e.store.UpsertSet(target, attrs)
		}
	}

	rows, err = e.store.LoadAll()
	if err != nil {
		return err
	}
	e.cache.Load(rows)
	return nil
}

// OnChange registers a listener that is called after any config change.
func (e *Engine) OnChange(fn NotifyFunc) {
	e.listeners = append(e.listeners, fn)
}

func (e *Engine) notify(target string) {
	for _, fn := range e.listeners {
		fn(target)
	}
}

// GetSet returns attrs for a SET target from cache.
func (e *Engine) GetSet(target string) map[string]string {
	return e.cache.GetSet(target)
}

// GetAdds returns all ADD instances for a target from cache.
func (e *Engine) GetAdds(target string) []RuntimeConfig {
	return e.cache.GetAdds(target)
}

// Exec parses and executes a command string.
func (e *Engine) Exec(cmd string) *ExecResult {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return &ExecResult{OK: false, Error: "empty command"}
	}

	parts := strings.SplitN(cmd, " ", 2)
	verb := strings.ToUpper(parts[0])
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
	}

	switch verb {
	case "ACT":
		return e.execAct(rest)
	case "SET":
		return e.execSet(rest)
	case "ADD":
		return e.execAdd(rest)
	case "LST":
		return e.execLst(rest)
	case "MOD":
		return e.execMod(rest)
	case "RMV":
		return e.execRmv(rest)
	default:
		return &ExecResult{OK: false, Error: "unknown command: " + verb}
	}
}

func (e *Engine) execAct(rest string) *ExecResult {
	switch strings.ToUpper(strings.TrimSpace(rest)) {
	case "SYSTEMRST":
		return &ExecResult{OK: true, Output: "system restarting..."}
	case "CLRLIMIT":
		e.notify("CLRLIMIT")
		return &ExecResult{OK: true, Output: "login rate limits cleared"}
	case "RELOAD":
		rows, err := e.store.LoadAll()
		if err != nil {
			return &ExecResult{OK: false, Error: err.Error()}
		}
		e.cache.Load(rows)
		for target := range map[string]bool{"SYSLOG": true, "JWT": true, "FILEUPLD": true, "LGFAILFIBPLCY": true, "CORS": true} {
			e.notify(target)
		}
		return &ExecResult{OK: true, Output: "config reloaded"}
	default:
		return &ExecResult{OK: false, Error: "unknown action: " + rest}
	}
}

func (e *Engine) execSet(rest string) *ExecResult {
	target, attrs, err := parseTargetAttrs(rest)
	if err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	if err := e.store.UpsertSet(target, attrs); err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}

	raw, _ := json.Marshal(attrs)
	e.cache.put(RuntimeConfig{Target: target, Kind: "set", AttrsJSON: string(raw)})
	e.notify(target)
	return &ExecResult{OK: true, Output: fmt.Sprintf("SET %s updated", target)}
}

func (e *Engine) execAdd(rest string) *ExecResult {
	target, attrs, err := parseTargetAttrs(rest)
	if err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	id, err := e.store.CreateAdd(target, attrs)
	if err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	raw, _ := json.Marshal(attrs)
	e.cache.put(RuntimeConfig{ID: id, Target: target, Kind: "add", AttrsJSON: string(raw)})
	e.notify(target)
	return &ExecResult{OK: true, Output: fmt.Sprintf("ADD %s created, ID=%d", target, id)}
}

func (e *Engine) execLst(rest string) *ExecResult {
	target := strings.TrimSpace(rest)
	if target == "" {
		return &ExecResult{OK: false, Error: "LST requires a target"}
	}

	// check for ID filter: "LGFAILFIBPLCY ID=1"
	targetPart := target
	var idFilter uint
	if idx := strings.Index(target, " "); idx > 0 {
		targetPart = target[:idx]
		filterPart := strings.TrimSpace(target[idx+1:])
		if strings.HasPrefix(strings.ToUpper(filterPart), "ID=") {
			idVal, _ := strconv.ParseUint(filterPart[3:], 10, 64)
			idFilter = uint(idVal)
		}
	}

	var rows []RuntimeConfig
	if idFilter > 0 {
		for _, r := range e.cache.GetAll() {
			if r.ID == idFilter && r.Target == targetPart {
				rows = append(rows, r)
				break
			}
		}
	} else {
		rows = e.cache.GetAdds(targetPart)
		if set := e.cache.GetSet(targetPart); set != nil {
			raw, _ := json.Marshal(set)
			rows = append(rows, RuntimeConfig{Target: targetPart, Kind: "set", AttrsJSON: string(raw)})
		}
	}

	if len(rows) == 0 {
		return &ExecResult{OK: true, Output: "(empty)"}
	}

	var sb strings.Builder
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		if r.Kind == "set" {
			sb.WriteString(fmt.Sprintf("[SET] %s:", r.Target))
		} else {
			sb.WriteString(fmt.Sprintf("[ADD] ID=%d %s:", r.ID, r.Target))
		}
		first := true
		for k, v := range attrs {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s=%s", k, v))
			first = false
		}
		sb.WriteString("\n")
	}
	return &ExecResult{OK: true, Output: strings.TrimSuffix(sb.String(), "\n")}
}

func (e *Engine) execMod(rest string) *ExecResult {
	target, attrs, err := parseTargetAttrs(rest)
	if err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	idStr, ok := attrs["ID"]
	if !ok {
		return &ExecResult{OK: false, Error: "MOD requires ID=<n>"}
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return &ExecResult{OK: false, Error: "invalid ID"}
	}
	delete(attrs, "ID")

	if err := e.store.UpdateAdd(uint(id), attrs); err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	raw, _ := json.Marshal(attrs)
	e.cache.put(RuntimeConfig{ID: uint(id), Target: target, Kind: "add", AttrsJSON: string(raw)})
	e.notify(target)
	return &ExecResult{OK: true, Output: fmt.Sprintf("MOD %s ID=%d updated", target, id)}
}

func (e *Engine) execRmv(rest string) *ExecResult {
	target, attrs, err := parseTargetAttrs(rest)
	if err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	idStr, ok := attrs["ID"]
	if !ok {
		return &ExecResult{OK: false, Error: "RMV requires ID=<n>"}
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return &ExecResult{OK: false, Error: "invalid ID"}
	}

	if err := e.store.DeleteAdd(uint(id)); err != nil {
		return &ExecResult{OK: false, Error: err.Error()}
	}
	e.cache.remove(uint(id), target)
	e.notify(target)
	return &ExecResult{OK: true, Output: fmt.Sprintf("RMV %s ID=%d deleted", target, id)}
}

func parseTargetAttrs(rest string) (string, map[string]string, error) {
	idx := strings.Index(rest, ":")
	if idx < 0 {
		return "", nil, errors.New("syntax: <TARGET>: <KEY=VALUE>[, ...]")
	}
	target := strings.TrimSpace(rest[:idx])
	if target == "" {
		return "", nil, errors.New("empty target")
	}

	attrsStr := strings.TrimSpace(rest[idx+1:])
	attrs := make(map[string]string)

	// parse comma-separated key=value pairs, handling values like "SINGLE_USER:testuser"
	pairs := splitAttrs(attrsStr)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return "", nil, fmt.Errorf("invalid key=value: %s", pair)
		}
		attrs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}

	return target, attrs, nil
}

func splitAttrs(s string) []string {
	var result []string
	var current strings.Builder
	inValue := false
	for _, ch := range s {
		switch ch {
		case ':':
			inValue = true
			current.WriteRune(ch)
		case ',':
			if inValue {
				current.WriteRune(ch)
			} else {
				result = append(result, current.String())
				current.Reset()
			}
		case ' ':
			if inValue {
				current.WriteRune(ch)
			} else {
				// skip leading spaces
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

// ExecResult is the result of a command execution.
type ExecResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SystemRestart triggers a process exit (Docker will restart).
func SystemRestart() {
	os.Exit(0)
}
