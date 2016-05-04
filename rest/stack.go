package rest

import (
	"sync"
	"errors"
)

type stack struct {
	lock sync.Mutex // you don't have to do this if you don't want thread safety
	s []interface{}
}

func NewStack() *stack {
	return &stack {sync.Mutex{}, make([]interface{},0), }
}

func (s *stack) Push(thing interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, thing)
}

func (s *stack) First() (interface{}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	l := len(s.s)
	if l == 0 {
		return nil, nil
	}
	return s.s[l-1], nil
}

func (s *stack) Pop() (interface{}, error) {
	s.lock.Lock()
	defer s.lock.Unlock()


	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Stack is empty")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}
