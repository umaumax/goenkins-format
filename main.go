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
	overwritFlag   bool
)

func init() {
	flag.IntVar(&indentSapceNum, "indent_num", 2, "number of spaces of indent")
	flag.BoolVar(&overwritFlag, "i", false, "Inplace edit <file>s, if specified. Don't use at /dev/stdin")
}

var (
	outputStream OutputStream
)

type OutputStream struct {
	output         string
	outputNewFlag  bool
	indentSapceNum int
}

func (s *OutputStream) Truncate() {
	s.output = ""
	s.outputNewFlag = false
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

	// NOTE: default input file is input pipe
	inputFiles := []string{"-"}
	if flag.NArg() > 0 {
		inputFiles = flag.Args()
	}
	completeNum := 0
	totalNum := len(inputFiles)
	for _, inputFile := range inputFiles {
		file := os.Stdin
		if inputFile != "-" {
			file, err := os.OpenFile(inputFile, os.O_RDWR, 0666)
			if err != nil {
				log.Println(err)
				continue
			}
			defer file.Close()
		}

		outputStream.Truncate()
		lexer := LexerWrapper{Lexer: NewLexer(file)}
		if yyParse(lexer) != 0 {
			log.Println(errors.New("hint fot error"))
			outputStream.TrimSpace()
			fmt.Fprintln(os.Stderr, "[", outputStream.output, "]")
			continue
		}

		outputStream.TrimSpace()

		if overwritFlag {
			if err := file.Truncate(0); err != nil {
				log.Println("Truncate:", err)
				continue
			} else if _, err := file.Seek(0, 0); err != nil {
				log.Println("Seek:", err)
				continue
			} else if _, err := fmt.Fprint(file, outputStream.output); err != nil {
				log.Println("Write:", err)
				continue
			}
		} else {
			fmt.Print(outputStream.output)
		}

		completeNum++
	}
	if completeNum != totalNum {
		fmt.Fprintf(os.Stderr, "failed processing (%d/%d)", totalNum-completeNum, totalNum)
		os.Exit(1)
	}
}
