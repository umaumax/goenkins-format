package main

import ()
import (
	"bufio"
	"io"
	"strings"
)

type frame struct {
	i            int
	s            string
	line, column int
}
type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch      chan frame
	ch_stop chan bool
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []frame
	stale bool

	// The 'l' and 'c' fields were added for
	// https://github.com/wagerlabs/docker/blob/65694e801a7b80930961d70c69cba9f2465459be/buildfile.nex
	// Since then, I introduced the built-in Line() and Column() functions.
	l, c int

	parseResult interface{}

	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan frame)
	yylex.ch_stop = make(chan bool, 1)
	var scan func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int)
	scan = func(in *bufio.Reader, ch chan frame, ch_stop chan bool, family []dfa, line, column int) {
		// Index of DFA and length of highest-precedence match so far.
		matchi, matchn := 0, -1
		var buf []rune
		n := 0
		checkAccept := func(i int, st int) bool {
			// Higher precedence match? DFAs are run in parallel, so matchn is at most len(buf), hence we may omit the length equality check.
			if family[i].acc[st] && (matchn < n || matchi > i) {
				matchi, matchn = i, n
				return true
			}
			return false
		}
		var state [][2]int
		for i := 0; i < len(family); i++ {
			mark := make([]bool, len(family[i].startf))
			// Every DFA starts at state 0.
			st := 0
			for {
				state = append(state, [2]int{i, st})
				mark[st] = true
				// As we're at the start of input, follow all ^ transitions and append to our list of start states.
				st = family[i].startf[st]
				if -1 == st || mark[st] {
					break
				}
				// We only check for a match after at least one transition.
				checkAccept(i, st)
			}
		}
		atEOF := false
		stopped := false
		for {
			if n == len(buf) && !atEOF {
				r, _, err := in.ReadRune()
				switch err {
				case io.EOF:
					atEOF = true
				case nil:
					buf = append(buf, r)
				default:
					panic(err)
				}
			}
			if !atEOF {
				r := buf[n]
				n++
				var nextState [][2]int
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if -1 == x[1] {
						continue
					}
					nextState = append(nextState, x)
					checkAccept(x[0], x[1])
				}
				state = nextState
			} else {
			dollar: // Handle $.
				for _, x := range state {
					mark := make([]bool, len(family[x[0]].endf))
					for {
						mark[x[1]] = true
						x[1] = family[x[0]].endf[x[1]]
						if -1 == x[1] || mark[x[1]] {
							break
						}
						if checkAccept(x[0], x[1]) {
							// Unlike before, we can break off the search. Now that we're at the end, there's no need to maintain the state of each DFA.
							break dollar
						}
					}
				}
				state = nil
			}

			if state == nil {
				lcUpdate := func(r rune) {
					if r == '\n' {
						line++
						column = 0
					} else {
						column++
					}
				}
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					lcUpdate(buf[0])
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1
					for {
						sent := false
						select {
						case ch <- frame{matchi, text, line, column}:
							{
								sent = true
							}
						case stopped = <-ch_stop:
							{
							}
						default:
							{
								// nothing
							}
						}
						if stopped || sent {
							break
						}
					}
					if stopped {
						break
					}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, ch_stop, family[matchi].nest, line, column)
					}
					if atEOF {
						break
					}
					for _, r := range text {
						lcUpdate(r)
					}
				}
				n = 0
				for i := 0; i < len(family); i++ {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		ch <- frame{-1, "", line, column}
	}
	go scan(bufio.NewReader(in), yylex.ch, yylex.ch_stop, dfas, 0, 0)
	return yylex
}

type dfa struct {
	acc          []bool           // Accepting states.
	f            []func(rune) int // Transitions.
	startf, endf []int            // Transitions at start and end of input.
	nest         []dfa
}

var dfas = []dfa{
	// [ \t]
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 32:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 9:
				return -1
			case 32:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \n
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// def|new|if|else|sh|echo|import|agent|label|script|environment|stage|node|dir|any|none|for|in
	{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, false, true, false, false, false, true, false, false, false, false, true, true, true, false, false, false, true, true, false, true, false, false, false, true, false, true, false, false, false, false, false, false, false, false, false, false, false, true, false, true, false, true, false, false, true, true, false, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 97:
				return 1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return 2
			case 101:
				return 3
			case 102:
				return 4
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 5
			case 108:
				return 6
			case 109:
				return -1
			case 110:
				return 7
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 8
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return 59
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 60
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 55
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 56
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return 39
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return 40
			case 109:
				return -1
			case 110:
				return 41
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 37
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return 30
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return 31
			case 110:
				return 32
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 26
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 19
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 20
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return 9
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return 10
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 11
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 15
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return 12
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return 13
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 14
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 16
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 17
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 18
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return 25
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return 21
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 22
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 24
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 23
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return 27
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 28
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return 29
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 33
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 34
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 35
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 36
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 38
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return 53
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 51
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return 42
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return 43
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 44
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 45
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 46
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return 47
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 48
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 49
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 50
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 52
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 54
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return 58
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 57
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 62
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return 61
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 63
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 64
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 118:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// ==|!=|>=|<=|\|\||&&
	{[]bool{false, false, false, false, false, false, false, true, true, true, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			case 38:
				return 2
			case 60:
				return 3
			case 61:
				return 4
			case 62:
				return 5
			case 124:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return 12
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return 11
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return 10
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return 9
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return 8
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 38:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 124:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// {|}
	{[]bool{false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 123:
				return 1
			case 125:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 123:
				return -1
			case 125:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 123:
				return -1
			case 125:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [;{}=+*%\/\-]|<|>
	{[]bool{false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 37:
				return 1
			case 42:
				return 1
			case 43:
				return 1
			case 45:
				return 1
			case 47:
				return 1
			case 59:
				return 1
			case 60:
				return 2
			case 61:
				return 1
			case 62:
				return 3
			case 123:
				return 1
			case 125:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			case 42:
				return -1
			case 43:
				return -1
			case 45:
				return -1
			case 47:
				return -1
			case 59:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			case 42:
				return -1
			case 43:
				return -1
			case 45:
				return -1
			case 47:
				return -1
			case 59:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 37:
				return -1
			case 42:
				return -1
			case 43:
				return -1
			case 45:
				return -1
			case 47:
				return -1
			case 59:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [(.]|\[
	{[]bool{false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 40:
				return 1
			case 46:
				return 1
			case 91:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 40:
				return -1
			case 46:
				return -1
			case 91:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 40:
				return -1
			case 46:
				return -1
			case 91:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

	// [:,]|\)|\]
	{[]bool{false, true, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 41:
				return 1
			case 44:
				return 2
			case 58:
				return 2
			case 93:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			case 44:
				return -1
			case 58:
				return -1
			case 93:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			case 44:
				return -1
			case 58:
				return -1
			case 93:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			case 44:
				return -1
			case 58:
				return -1
			case 93:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// [a-zA-Z0-9$_]+
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 36:
				return 1
			case 95:
				return 1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			case 65 <= r && r <= 90:
				return 1
			case 97 <= r && r <= 122:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 36:
				return 1
			case 95:
				return 1
			}
			switch {
			case 48 <= r && r <= 57:
				return 1
			case 65 <= r && r <= 90:
				return 1
			case 97 <= r && r <= 122:
				return 1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \/\/[^\n]*\n
	{[]bool{false, false, false, true, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 47:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return 3
			case 47:
				return 4
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 10:
				return -1
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 10:
				return 3
			case 47:
				return 4
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// \/\*([^\/]|[^*]\/)*\*\/
	{[]bool{false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 2
			case 47:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 8
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 7
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 6
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return 3
			case 47:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 42:
				return -1
			case 47:
				return 7
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// '''([^']|'[^']|''[^'])*'''
	{[]bool{false, false, false, false, false, false, false, false, true, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 39:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 39:
				return 8
			}
			return 9
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			}
			return 5
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// """([^"]|"[^"]|""[^"])*"""
	{[]bool{false, false, false, false, false, false, false, false, true, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 34:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 34:
				return 6
			}
			return 7
		},
		func(r rune) int {
			switch r {
			case 34:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 34:
				return 8
			}
			return 9
		},
		func(r rune) int {
			switch r {
			case 34:
				return 4
			}
			return 5
		},
		func(r rune) int {
			switch r {
			case 34:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 4
			}
			return 5
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// '[^']*'
	{[]bool{false, false, true, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			}
			return 3
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

	// "([^"]|\\[^"])*"
	{[]bool{false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 34:
				return 1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 34:
				return -1
			case 92:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 5
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 5
			}
			return 6
		},
		func(r rune) int {
			switch r {
			case 34:
				return 2
			case 92:
				return 3
			}
			return 4
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},
}

func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}

func (yyLex *Lexer) Stop() {
	yyLex.ch_stop <- true
}

// Text returns the matched text.
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}

// Line returns the current line number.
// The first line is 0.
func (yylex *Lexer) Line() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].line
}

// Column returns the current column number.
// The first column is 0.
func (yylex *Lexer) Column() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].column
}

func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		l, c := 0, 0
		if lvl > 0 {
			l, c = yylex.stack[lvl-1].line, yylex.stack[lvl-1].column
		}
		yylex.stack = append(yylex.stack, frame{0, "", l, c})
	}
	if lvl == len(yylex.stack)-1 {
		p := &yylex.stack[lvl]
		*p = <-yylex.ch
		yylex.stale = false
	} else {
		yylex.stale = true
	}
	return yylex.stack[lvl].i
}
func (yylex *Lexer) pop() {
	yylex.stack = yylex.stack[:len(yylex.stack)-1]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}

// Lex runs the lexer. Always returns 0.
// When the -s option is given, this function is not generated;
// instead, the NN_FUN macro runs the lexer.
func (yylex *Lexer) Lex(lval *yySymType) int {
OUTER0:
	for {
		switch yylex.next(0) {
		case 0:
			{ /* skip */
			}
		case 1:
			{
				outputStream.TrimSpace()
				outputStream.Write(0, "\n")
				outputStream.SetNewLineFlag()
				return NR
			}
		case 2:
			{
				m := map[string]int{
					"def":         DEF,
					"new":         NEW,
					"if":          IF,
					"else":        ELSE,
					"sh":          SH,
					"echo":        ECHO,
					"import":      IMPORT,
					"agent":       AGENT,
					"label":       LABEL,
					"script":      SCRIPT,
					"environment": ENVIRONMENT,
					"stage":       STAGE,
					"node":        NODE,
					"dir":         DIR,
					"any":         ANY,
					"none":        NONE,
					"for":         FOR,
					"in":          IN,
				}
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return m[yylex.Text()]
			}
		case 3:
			{
				m := map[string]int{
					"==": EQ,
					"!=": NE,
					">=": GE,
					"<=": LE,
					"||": OR,
					"&&": AND,
				}
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return m[yylex.Text()]
			}
		case 4:
			{
				c := yylex.Text()[0]
				if c == '{' {
					lval.indent_level++
				}
				if c == '}' {
					lval.indent_level--
				}
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return int(c)
			}
		case 5:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return int(yylex.Text()[0])
			}
		case 6:
			{
				c := yylex.Text()[0]
				if c == '(' || c == '[' {
					lval.indent_level++
				}
				outputStream.TrimSpace()
				outputStream.Write(lval.indent_level, yylex.Text())
				return int(c)
			}
		case 7:
			{
				c := yylex.Text()[0]
				if c == ')' || c == ']' {
					lval.indent_level--
				}
				outputStream.TrimSpace()
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return int(yylex.Text()[0])
			}
		case 8:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				lval.str = yylex.Text()
				return IDENT
			}
		case 9:
			{
				outputStream.Write(lval.indent_level, yylex.Text())
				outputStream.SetNewLineFlag()
				// NOTE: return new line value because of including \n at the end
				return NR
			}
		case 10:
			{
				outputStream.Write(lval.indent_level, yylex.Text())
				// NOTE: skipe multi line comment
			}
		case 11:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				lval.str = yylex.Text()
				return STRING
			}
		case 12:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				lval.str = yylex.Text()
				return STRING
			}
		case 13:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return STRING
			}
		case 14:
			{
				outputStream.Write(lval.indent_level, yylex.Text(), " ")
				return STRING
			}
		default:
			break OUTER0
		}
		continue
	}
	yylex.pop()

	return 0
}
