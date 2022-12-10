package utilities

import (
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	ErrInvalidCharacter = errors.New("invalid character")
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

// The rule describes the transtion from each state to another. The transition
// is performed with every new character.
type Rule struct {
	FromState  State      // Current state
	Symbols    []byte     // Any of these symbols will trigger this transition
	ToState    State      // Target state of transition
	ActionCode ActionCode // Define an action to perform when this rule is used
	Scan       bool       // If enabled the character is added to the buffer
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

// Helper function to shorten strings to 15 characters for log-printing
func prettyprint(datain []byte) string {
	fewbytes := datain
	if len(datain) > 15 {
		fewbytes = datain[:10]
	}
	str := makeBytesReadable(fewbytes)
	if len(datain) > 15 {
		str = str + "..."
	}
	return str
}

func (s *fsm) Push(token byte) ([]byte, ActionCode, error) {

	rule, err := s.findMatchingRule(token)

	if os.Getenv("PROTOLOG_ENABLE") == "extended" {
		fmt.Printf(" F|%s| %3d -> %3d with token 0x%02x '%s' [%d->%d, Symbols:'%s', ActionCode:'%s', scan:%t]\n",
			time.Now().Format("20060102 150405.0"),
			s.currentState,
			rule.ToState,
			token,
			makeBytesReadable([]byte{token}),
			rule.FromState,
			rule.ToState,
			prettyprint(rule.Symbols),
			rule.ActionCode,
			rule.Scan)
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
	return Rule{}, fmt.Errorf(`%w : "%s" ascii: %q currentBuffer: "%s" , status of fsm: %d`, ErrInvalidCharacter, string(token), token, string(s.currentBuffer), s.currentState)
}

func (s *fsm) Init() {
	s.currentState = Init
}
