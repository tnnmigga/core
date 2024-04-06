package core

// 信号量 PV操作
type Semaphore struct {
	count chan struct{}
}

func NewSemaphore(n int) *Semaphore {
	return &Semaphore{count: make(chan struct{}, n)}
}

func (s *Semaphore) P() {
	s.count <- struct{}{}
}

func (s *Semaphore) V() {
	<-s.count
}
