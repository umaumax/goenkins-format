package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

var output string
var outputNewFlag bool

func TrimSpace() {
	output = strings.TrimRight(output, " ")
	if strings.HasSuffix(output, "\n") {
		output = strings.TrimRight(output, "\n") + "\n"
	}
}
func Output(indent_level int, args ...interface{}) {
	if outputNewFlag {
		output += fmt.Sprint(GenIndent(indent_level))
		outputNewFlag = false
	}
	withSpaceKeywords := []string{"if", ":"}
	for _, v := range withSpaceKeywords {
		if strings.HasSuffix(output, v) {
			output += " "
			break
		}
	}
	output += fmt.Sprint(args...)
}
func GenIndent(indent_level int) string {
	// NOTE: indent space number is 2
	return strings.Repeat("  ", indent_level)
}

func main() {
	if yyParse(NewLexer(os.Stdin)) != 0 {
		log.Fatal(errors.New("Parse error"))
	}
	TrimSpace()
	fmt.Print(output)
}
