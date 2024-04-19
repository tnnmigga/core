package process

import (
	"os"
	"os/signal"
	"syscall"
)

var (
	exitSignals = []os.Signal{syscall.SIGQUIT, os.Interrupt, syscall.SIGTERM}
	sign        = make(chan os.Signal, 1)
)

func WaitExitSignal() os.Signal {
	signal.Notify(sign, exitSignals...)
	return <-sign
}

// 运行时故障触发进程退出流程
func Exit() {
	select {
	case sign <- syscall.SIGQUIT:
	default:
	}
}
