package idef

type ServerState int

const (
	ServerStateInit ServerState = iota // 进程启动后初始化模块/日志/配置等工作
	ServerStateRun                     // 运行各个模块
	ServerStateStop                    // 停止各个模块
	ServerStateExit                    // 进程退出阶段
)

const (
	ConstKeyNone         = "none"
	ConstKeyNonuseStream = "nonuse-stream"
	ConstKeyOneOfMods    = "one-of-mods"
	ConstKeyServerID     = "server-id"
	ConstKeyExpires      = "expires"
)

type ModName string

const (
	ModLink ModName = "link"
)
