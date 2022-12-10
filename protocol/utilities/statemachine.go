package utilities

import (
	"errors"
	"fmt"
	"os"
	"time"
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

var ASCIIMap = map[byte]string{
	0:  "<NUL>",
	1:  "<SOH>",
	2:  "<STX>",
	3:  "<ETX>",
	4:  "<EOT>",
	5:  "<ENQ>",
	6:  "<ACK>",
	7:  "<BEL>",
	8:  "<BS>",
	9:  "<HT>",
	10: "<LF>",
	11: "<VT>",
	12: "<FF>",
	13: "<CR>",
	14: "<SO>",
	15: "<SI>",
	16: "<DLE>",
	17: "<DC1>",
	18: "<DC2>",
	19: "<DC3>",
	20: "<DC4>",
	21: "<NAK>",
	22: "<SYN>",
	23: "<ETB>",
	24: "<CAN>",
	25: "<EM>",
	26: "<SUB>",
	27: "<ESC>",
	28: "<FS>",
	29: "<GS>",
	30: "<RS>",
	31: "<US>",
}

func makeBytesReadable(in []byte) string {
	ret := ""
	for i := 0; i < len(in); i++ {
		if in[i] < 32 {
			ret = ret + ASCIIMap[in[i]]
		} else {
			ret = ret + string(in[i])
		}
	}
	return ret
}

func (s *fsm) Push(token byte) ([]byte, ActionCode, error) {

	rule, err := s.findMatchingRule(token)

	if os.Getenv("PROTOLOG_ENABLE") == "extended" {
		fmt.Printf(" F|%s| %3d -> %3d with token 0x%02x '%s' (rule:%#v)\n", time.Now().Format("20060102 150405.0"), s.currentState, rule.ToState, token, makeBytesReadable([]byte{token}), rule)
	}

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
