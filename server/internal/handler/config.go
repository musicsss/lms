package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lms/server/internal/runtimecfg"
)

// ConfigHandler 处理运行时配置相关的 HTTP 请求。
type ConfigHandler struct {
	engine *runtimecfg.Engine
}

func NewConfigHandler(engine *runtimecfg.Engine) *ConfigHandler {
	return &ConfigHandler{engine: engine}
}

// Exec 执行运行时配置命令（SET / ADD / LST / MOD / RMV / ACT）。
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

	// SYSTEMRST 特殊处理：返回响应后触发进程退出
	if input.Command == "ACT SYSTEMRST" && result.OK {
		go runtimecfg.SystemRestart()
	}
}

// ============================================================
// 以下类型定义用于 GET /admin/config/targets 响应的序列化
// ============================================================

type targetField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // select / number / text
	Options     []string `json:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Min         int      `json:"min,omitempty"`
	Max         int      `json:"max,omitempty"`
}

type targetMeta struct {
	Target    string            `json:"target"`
	Kind      string            `json:"kind"`
	Label     string            `json:"label"`
	Fields    []targetField     `json:"fields"`
	Value     map[string]string `json:"value,omitempty"`
	Instances []instanceView    `json:"instances,omitempty"`
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

// Targets 返回前端配置页面所需的全部可配置目标元数据。
func (h *ConfigHandler) Targets(c *gin.Context) {
	categories := []categoryMeta{
		{
			Name: "登录保护",
			Targets: []targetMeta{
				{
					Target: runtimecfg.TargetLoginFail,
					Kind:   runtimecfg.KindAdd,
					Label:  "登录失败封禁策略",
					Fields: []targetField{
						{Key: runtimecfg.FieldRange, Label: "适用范围", Type: "select", Options: []string{"ALL_USER", "SINGLE_USER", "IP"}},
						{Key: runtimecfg.FieldBlockPolicy, Label: "封禁粒度", Type: "select", Options: []string{"ACCOUNT", "IP"}},
					},
					Instances: buildInstances(h.engine.GetAdds(runtimecfg.TargetLoginFail)),
				},
			},
		},
		{
			Name: "系统参数",
			Targets: []targetMeta{
				{
					Target: runtimecfg.TargetSyslog,
					Kind:   runtimecfg.KindSet,
					Label:  "系统日志等级",
					Fields: []targetField{
						{Key: runtimecfg.FieldLevel, Label: "日志等级", Type: "select", Options: []string{"DEBUG", "INFO", "WARN", "ERROR"}},
					},
					Value: h.engine.GetSet(runtimecfg.TargetSyslog),
				},
				{
					Target: runtimecfg.TargetJWT,
					Kind:   runtimecfg.KindSet,
					Label:  "JWT 过期时间",
					Fields: []targetField{
						{Key: runtimecfg.FieldExpireTime, Label: "过期时间（小时）", Type: "number", Min: 1, Max: 720},
					},
					Value: h.engine.GetSet(runtimecfg.TargetJWT),
				},
				{
					Target: runtimecfg.TargetFileUpl,
					Kind:   runtimecfg.KindSet,
					Label:  "文件上传限制",
					Fields: []targetField{
						{Key: runtimecfg.FieldMaxSize, Label: "最大上传大小（MB）", Type: "number", Min: 1, Max: 10240},
					},
					Value: h.engine.GetSet(runtimecfg.TargetFileUpl),
				},
			},
		},
		{
			Name: "安全",
			Targets: []targetMeta{
				{
					Target: runtimecfg.TargetCORS,
					Kind:   runtimecfg.KindAdd,
					Label:  "CORS 白名单",
					Fields: []targetField{
						{Key: runtimecfg.FieldOrigin, Label: "允许来源", Type: "text", Placeholder: "http://example.com"},
					},
					Instances: buildInstances(h.engine.GetAdds(runtimecfg.TargetCORS)),
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

// buildInstances 将 RuntimeConfig 列表转换为前端可展示的实例视图。
func buildInstances(rows []runtimecfg.RuntimeConfig) []instanceView {
	result := make([]instanceView, 0, len(rows))
	for _, r := range rows {
		var attrs map[string]string
		json.Unmarshal([]byte(r.AttrsJSON), &attrs)
		result = append(result, instanceView{ID: r.ID, Attrs: attrs})
	}
	return result
}
