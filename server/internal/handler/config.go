package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/runtimecfg"
)

type ConfigHandler struct {
	engine *runtimecfg.Engine
}

func NewConfigHandler(engine *runtimecfg.Engine) *ConfigHandler {
	return &ConfigHandler{engine: engine}
}

func (h *ConfigHandler) Exec(c *gin.Context) {
	var input struct {
		Command string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.engine.Exec(input.Command)
	c.JSON(http.StatusOK, result)

	// special: if SYSTEMRST was executed, trigger restart after response
	if input.Command == "ACT SYSTEMRST" && result.OK {
		go runtimecfg.SystemRestart()
	}
}

type targetField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // select, number, text
	Options     []string `json:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Min         int      `json:"min,omitempty"`
	Max         int      `json:"max,omitempty"`
}

type targetMeta struct {
	Target   string                `json:"target"`
	Kind     string                `json:"kind"`
	Label    string                `json:"label"`
	Fields   []targetField         `json:"fields"`
	Value    map[string]string     `json:"value,omitempty"`
	Instances []instanceView        `json:"instances,omitempty"`
}

type instanceView struct {
	ID    uint              `json:"id"`
	Attrs map[string]string `json:"attrs"`
}

type categoryMeta struct {
	Name    string       `json:"name"`
	Targets []targetMeta `json:"targets"`
}

type actionMeta struct {
	Action  string `json:"action"`
	Label   string `json:"label"`
	Confirm string `json:"confirm,omitempty"`
}

func (h *ConfigHandler) Targets(c *gin.Context) {
	categories := []categoryMeta{
		{
			Name: "登录保护",
			Targets: []targetMeta{
				{
					Target: "LGFAILFIBPLCY",
					Kind:   "add",
					Label:  "登录失败封禁策略",
					Fields: []targetField{
						{Key: "RANGE", Label: "适用范围", Type: "select", Options: []string{"ALL_USER", "SINGLE_USER", "IP"}},
						{Key: "BLOCKPLCY", Label: "封禁粒度", Type: "select", Options: []string{"ACCOUNT", "IP"}},
					},
					Instances: buildInstances(h.engine.GetAdds("LGFAILFIBPLCY")),
				},
			},
		},
		{
			Name: "系统参数",
			Targets: []targetMeta{
				{
					Target: "SYSLOG",
					Kind:   "set",
					Label:  "系统日志等级",
					Fields: []targetField{
						{Key: "LEVEL", Label: "日志等级", Type: "select", Options: []string{"DEBUG", "INFO", "WARN", "ERROR"}},
					},
					Value: h.engine.GetSet("SYSLOG"),
				},
				{
					Target: "JWT",
					Kind:   "set",
					Label:  "JWT 过期时间",
					Fields: []targetField{
						{Key: "EXPIRETIME", Label: "过期时间（小时）", Type: "number", Min: 1, Max: 720},
					},
					Value: h.engine.GetSet("JWT"),
				},
				{
					Target: "FILEUPLD",
					Kind:   "set",
					Label:  "文件上传限制",
					Fields: []targetField{
						{Key: "MAXSIZE", Label: "最大上传大小（MB）", Type: "number", Min: 1, Max: 10240},
					},
					Value: h.engine.GetSet("FILEUPLD"),
				},
			},
		},
		{
			Name: "安全",
			Targets: []targetMeta{
				{
					Target: "CORS",
					Kind:   "add",
					Label:  "CORS 白名单",
					Fields: []targetField{
						{Key: "ORIGIN", Label: "允许来源", Type: "text", Placeholder: "http://example.com"},
					},
					Instances: buildInstances(h.engine.GetAdds("CORS")),
				},
			},
		},
	}

	actions := []actionMeta{
		{Action: "SYSTEMRST", Label: "重启系统", Confirm: "确认重启系统？服务将短暂中断。"},
		{Action: "CLRLIMIT", Label: "清除登录限流", Confirm: "确认清除所有登录限流计数？"},
		{Action: "RELOAD", Label: "重载配置", Confirm: ""},
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"actions":    actions,
	})
}

func buildInstances(rows []runtimecfg.RuntimeConfig) []instanceView {
	result := make([]instanceView, 0, len(rows))
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		result = append(result, instanceView{ID: r.ID, Attrs: attrs})
	}
	return result
}
