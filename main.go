package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	indentSapceNum int
)

func init() {
	flag.IntVar(&indentSapceNum, "indent_num", 2, "number of spaces of indent")
}

var (
	outputStream OutputStream
)

type OutputStream struct {
	output         string
	outputNewFlag  bool
	indentSapceNum int
}

func (s *OutputStream) SetIndentSpaceNum(indentSapceNum int) {
	s.indentSapceNum = indentSapceNum
}

func (s *OutputStream) SetNewLineFlag() {
	s.outputNewFlag = true
}

func (s *OutputStream) TrimSpace() {
	s.output = strings.TrimRight(s.output, " ")
	if strings.HasSuffix(s.output, "\n") {
		s.output = strings.TrimRight(s.output, "\n") + "\n"
	}
}
func (s *OutputStream) Write(indent_level int, args ...interface{}) {
	if s.outputNewFlag {
		s.output += fmt.Sprint(s.genIndent(indent_level))
		s.outputNewFlag = false
	}
	withSpaceKeywords := []string{"if", ":"}
	for _, v := range withSpaceKeywords {
		if strings.HasSuffix(s.output, v) {
			s.output += " "
			break
		}
	}
	s.output += fmt.Sprint(args...)
}
func (s *OutputStream) genIndent(indent_level int) string {
	return strings.Repeat(strings.Repeat(" ", s.indentSapceNum), indent_level)
}

func (s *OutputStream) String() string {
	return s.output
}

type LexerWrapper struct {
	*Lexer
}

func (yylex LexerWrapper) Error(e string) {
	log.Println(e)
}

func main() {
	flag.Parse()

	outputStream.SetIndentSpaceNum(indentSapceNum)

	lexer := LexerWrapper{Lexer: NewLexer(os.Stdin)}
	if yyParse(lexer) != 0 {
		log.Println(errors.New("hint fot error"))
		outputStream.TrimSpace()
		fmt.Fprintln(os.Stderr, "[", outputStream.output, "]")
		os.Exit(1)
	}

	outputStream.TrimSpace()
	fmt.Print(outputStream.output)
}
