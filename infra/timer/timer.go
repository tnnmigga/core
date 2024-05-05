package timer

import (
	"context"
	"fmt"

	"github.com/tnnmigga/core/algorithm"
	"github.com/tnnmigga/core/conc"
	"github.com/tnnmigga/core/idef"
	"github.com/tnnmigga/core/msgbus"
	"github.com/tnnmigga/core/utils"
	"github.com/tnnmigga/core/utils/idgen"

	"time"
)

type timerTrigger struct {
}

type timerCtx struct {
	ID   uint64
	Time time.Duration
	Ctx  any
}

func (t *timerCtx) String() string {
	return fmt.Sprintf("{ID: %d, Time: %d, Body: %s}", t.ID, t.Time, utils.String(t.Ctx))
}

func (t *timerCtx) Key() uint64 {
	return t.ID
}

func (t *timerCtx) Value() time.Duration {
	return t.Time
}

type TimerHeap struct {
	algorithm.Heap[uint64, time.Duration, *timerCtx]
	module idef.IModule
	timer  *time.Timer
}

func NewTimerHeap(m idef.IModule) *TimerHeap {
	h := &TimerHeap{
		module: m,
	}
	msgbus.RegisterHandler(m, h.onTimerTrigger)
	return h
}

func (h *TimerHeap) New(delay time.Duration, ctx any) uint64 {
	t := &timerCtx{
		ID:   idgen.NewUUID(),
		Time: utils.NowNs() + delay,
		Ctx:  ctx,
	}
	top := h.Top()
	h.Push(t)
	if top != nil && top.Time <= t.Time {
		return t.ID
	}
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
	h.tryNextTrigger()
	return t.ID
}

func (h *TimerHeap) Stop(id uint64) bool {
	index := h.Find(id)
	if index == -1 {
		return false
	}
	h.RemoveByIndex(index)
	return true
}

func (h *TimerHeap) tryNextTrigger() {
	top := h.Top()
	if top == nil {
		return
	}
	nowNs := utils.NowNs()
	if top.Time <= nowNs {
		h.trigger()
		return
	}
	h.timer = time.NewTimer(time.Duration(top.Time - utils.NowNs()))
	conc.Go(func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-h.timer.C:
			h.trigger()
		}
	})
}

func (h *TimerHeap) trigger() {
	// 此函数的执行协程为模块协程外的临时开辟的协程
	// 若直接操作数据会存在并发问题
	// 因此是投递消息给模块由模块协程来操作定时器数据
	h.module.Assign(&timerTrigger{})
}
