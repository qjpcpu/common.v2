package joint

import (
	"log"
	"reflect"
	"sync/atomic"
	"time"
)

const (
	reloadI = iota
	timeoutI
	closeI
	readI
	writeI
)

type scheduler struct {
	*Joint
	term                 time.Duration
	timer                *time.Timer
	readChannels         []reflect.SelectCase
	readAndWriteChannels []reflect.SelectCase
	dummyC               reflect.Value // dummy channel for place holder
	lastE                interface{}   // last enqueue value
	lastD                interface{}   // last dequeue value
	aborted              bool
	inputClosed          bool
}

func newScheduler(j *Joint) *scheduler {
	// add timer to prevent fatal error: all goroutines are asleep - deadlock!
	term := time.Hour * 1
	timer := time.NewTimer(term)
	readChannels := []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(j.reloadC), // reload channel
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(timer.C),
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(j.breakC),
		},
		{
			Dir:  reflect.SelectRecv,
			Chan: j.readC,
		},
	}
	readAndWriteChannels := append(readChannels, reflect.SelectCase{
		Dir:  reflect.SelectSend,
		Chan: j.writeC,
	})
	dummyC := reflect.ValueOf(make(chan struct{}, 1))
	return &scheduler{
		Joint:                j,
		term:                 term,
		readChannels:         readChannels,
		readAndWriteChannels: readAndWriteChannels,
		dummyC:               dummyC,
		timer:                timer,
	}
}

func (s *scheduler) resetTimer() {
	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}
	s.timer.Reset(s.term)
}

func (s *scheduler) stop() {
	s.timer.Stop()
}

func (s *scheduler) isAborted() bool { return s.aborted }

func (s *scheduler) runOnce() {
	s.resetTimer()
	if s.Joint.queueSize == 0 {
		s.waitRead()
	} else {
		s.waiteReadOrWrite()
	}
}

func (s *scheduler) waitRead() {
	if s.inputClosed {
		s.aborted = true
		return
	}
	// list is empty
	chosen, recv, ok := reflect.Select(s.readChannels)
	if !ok {
		s.aborted = true
		return
	}
	if chosen == timeoutI || chosen == reloadI {
		return
	}
	dataVal := recv.Interface()
	// drop data by filter
	if s.Joint.filter != nil && !s.Joint.filter(dataVal) {
		return
	}
	s.Joint.queueSize++
	s.readAndWriteChannels[writeI].Send = recv
	if Debug {
		s.lastE = recv.Interface()
		s.lastD = recv.Interface()
		log.Printf("[joint] Enqueue %v", s.lastE)
	}
}

func (s *scheduler) waiteReadOrWrite() {
	chosen, recv, ok := s.tryReadOrWrite()
	if chosen == timeoutI || chosen == reloadI {
		return
	}
	if chosen == writeI {
		// write success
		s.handleSend(chosen, recv)
	} else {
		// read a signal
		s.handleRecv(chosen, recv, ok)
	}
}

func (s *scheduler) tryReadOrWrite() (chosen int, recv reflect.Value, ok bool) {
	if buff := atomic.LoadUint64(&s.Joint.maxIn); s.Joint.queueSize >= buff {
		// block read channel
		s.readAndWriteChannels[readI].Chan = s.dummyC
		chosen, recv, ok = reflect.Select(s.readAndWriteChannels)
		// restore readC
		s.readAndWriteChannels[readI].Chan = s.Joint.readC
	} else {
		chosen, recv, ok = reflect.Select(s.readAndWriteChannels)
	}
	return
}

func (s *scheduler) prepareNextWrite() {
	for s.Joint.queueSize > 0 {
		val, _ := s.Joint.list.pop()
		if s.Joint.filter != nil && !s.Joint.filter(val.Interface()) {
			s.Joint.queueSize--
		} else {
			s.readAndWriteChannels[writeI].Send = val
			if Debug {
				s.lastD = val.Interface()
			}
			break
		}
	}
}

func (s *scheduler) handleSend(chosen int, recv reflect.Value) {
	s.Joint.queueSize--
	if Debug {
		log.Printf("[joint] Dequeue %v", s.lastD)
	}
	s.prepareNextWrite()
}

func (s *scheduler) handleRecv(chosen int, recv reflect.Value, ok bool) {
	if !ok {
		if chosen == closeI {
			s.aborted = true
			return
		} else {
			if Debug {
				log.Println("[joint] Input channel closed.")
			}
			// do not read from input channel any more
			s.readChannels[readI].Chan = s.dummyC
			s.readAndWriteChannels[readI].Chan = s.dummyC
			s.inputClosed = true
			return
		}
	}
	if chosen == readI {
		// read ok
		s.Joint.list.push(recv)
		s.Joint.queueSize++
		if Debug {
			s.lastE = recv.Interface()
			log.Printf("[joint] Enqueue %v", s.lastE)
		}
	}
}
