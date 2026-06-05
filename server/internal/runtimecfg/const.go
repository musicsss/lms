package runtimecfg

// 运行时配置中已注册的 target 名称
const (
	TargetSyslog    = "SYSLOG"        // 系统日志等级
	TargetJWT       = "JWT"           // JWT 过期时间
	TargetFileUpl   = "FILEUPLD"      // 文件上传限制
	TargetLoginFail = "LGFAILFIBPLCY" // 登录封禁策略
	TargetCORS      = "CORS"          // CORS 白名单
)

// 配置项的字段名
const (
	FieldLevel       = "LEVEL"      // 日志等级值
	FieldExpireTime  = "EXPIRETIME" // JWT 过期小时
	FieldMaxSize     = "MAXSIZE"    // 文件上传上限 (MB)
	FieldRange       = "RANGE"      // 封禁适用范围
	FieldBlockPolicy = "BLOCKPLCY"  // 封禁粒度
	FieldOrigin      = "ORIGIN"     // CORS 来源
)

// 配置存储类型
const (
	KindSet = "set" // 单值配置 (SET)
	KindAdd = "add" // 多实例配置 (ADD)
)

// AllTargets 返回运行时配置引擎需要监听变更的 target 列表。
func AllTargets() []string {
	return []string{TargetSyslog, TargetJWT, TargetFileUpl, TargetLoginFail, TargetCORS}
}
