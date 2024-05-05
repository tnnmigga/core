package timer

import (
	"github.com/tnnmigga/core/msgbus"
	"github.com/tnnmigga/core/utils"
)

func (h *TimerHeap) onTimerTrigger(msg *timerTrigger) {
	defer h.tryNextTrigger()
	nowNs := utils.NowNs()
	for top := h.Top(); top != nil && top.Time <= nowNs; top = h.Top() {
		h.Pop()
		msgbus.Cast(top.Ctx)
	}
}
