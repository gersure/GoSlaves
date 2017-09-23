package slaves

import (
	"sync"
	"sync/atomic"
)

type work struct {
	work      func(interface{}) interface{}
	afterWork func(interface{})
}

type slave struct {
	opened  bool
	ready   int32
	jobChan chan interface{}
	mx      sync.Mutex
	wg      sync.WaitGroup
	Owner   *SlavePool
	work    *work
	Type    []byte
}

// Open Starts the slave creating goroutine
// that waits job notification
func (s *slave) Open() error {
	if s.work == nil {
		return errworkIsNil
	}
	s.opened = true
	s.ready = 1
	s.jobChan = make(chan interface{})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Loop until jobChan is closed
		for data := range s.jobChan {
			atomic.StoreInt32(&s.ready, 0)

			ret := s.work.work(data)
			if s.work.afterWork != nil {
				s.work.afterWork(ret)
			}
			s.Owner.wg.Add(-1)

			// notify slave is ready to work
			atomic.StoreInt32(&s.ready, 1)
		}
	}()

	return nil
}

// SetWork sets new Work for slave.
// If toDo is nil, the parameter is ignored
// it's not the same with afterWork value, because this is not important
func (s *slave) SetWork(
	toDo func(interface{}) interface{},
	afterWork func(interface{}),
) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.work == nil {
		s.work = new(work)
	}

	if toDo != nil {
		s.work.work = toDo
	}
	s.work.afterWork = afterWork
}

// Close the slave waiting to finish his tasks
func (s *slave) Close() {
	close(s.jobChan)
	s.opened = false
	s.wg.Wait()
}
