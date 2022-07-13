package utilities

import (
	"errors"
	"fmt"
)

var (
	InvalidCharacterError = errors.New("invalid character")
)

type ActionCode string

const (
	Consumed       ActionCode = ""
	Ok             ActionCode = ""
	Start          ActionCode = "Start"
	CheckSum       ActionCode = "CheckSum"
	ProcessMessage ActionCode = "ProcessMessage"
	Error          ActionCode = "Error"
	Finish         ActionCode = "Finish"
)

type State int

const (
	Init State = iota
)

// Rule struct for Rule
type Rule struct {
	FromState  State
	Symbols    []byte
	ToState    State
	ActionCode ActionCode
	Scan       bool
}

type FiniteStateMachine interface {
	Push(token byte) ([]byte, ActionCode, error)
	ResetBuffer()
	Init()
}

type fsm struct {
	rules         []Rule
	currentState  State
	currentBuffer []byte
}

func CreateFSM(data []Rule) FiniteStateMachine {
	return &fsm{
		rules:         data,
		currentBuffer: make([]byte, 0),
		currentState:  Init,
	}
}

func (s *fsm) Push(token byte) ([]byte, ActionCode, error) {
	rule, err := s.findMatchingRule(token)
	if err != nil {
		return nil, Error, err
	}

	if rule.Scan {
		s.currentBuffer = append(s.currentBuffer, token)
	}

	s.currentState = rule.ToState

	if rule.ActionCode != Consumed {
		return s.currentBuffer, rule.ActionCode, nil
	}

	return []byte{}, Consumed, nil
}

func (s *fsm) ResetBuffer() {
	s.currentBuffer = make([]byte, 0)
}

func (s *fsm) findMatchingRule(token byte) (Rule, error) {
	for _, rule := range s.rules {
		if rule.FromState == s.currentState && s.containsSymbol(rule.Symbols, token) {
			return rule, nil
		}
	}
	return Rule{}, fmt.Errorf(`%w : "%s" ascii: %q currentBuffer: "%s" , status of fsm: %d`, InvalidCharacterError, string(token), token, string(s.currentBuffer), s.currentState)
}

func (s *fsm) Init() {
	s.currentState = Init
}

func (s *fsm) containsSymbol(symbols []byte, symbol byte) bool {
	for _, x := range symbols {
		if x == symbol {
			return true
		}
	}
	return false
}
