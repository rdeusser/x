package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

var method string

type methodSignature struct {
	name     string
	receiver string
}

func main() {
	flag := flag.NewFlagSet("add-method", flag.ContinueOnError)

	flag.StringVar(&method, "method", "", "method name")

	if err := flag.Parse(os.Args[1:]); err != nil {
		log.Fatalf("%+v", err)
	}

	args := flag.Args()
	if len(args) != 1 {
		log.Fatalf("only one file may be passed at a time")
	}

	filename := args[0]
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	buf := bytes.Buffer{}
	lines := getLines(data)

	for _, line := range lines {
		if strings.Contains(line, "func") && strings.Contains(line, method) {
			sig := parseMethodSignature(line)

			buf.WriteString("func (")
			buf.WriteString(sig.name)
			buf.WriteString(" ")
			buf.WriteString(sig.receiver)
			buf.WriteString(") ")
			buf.WriteString(method)
			buf.WriteString("AddMethod")
			buf.WriteString("() string ")
			buf.WriteString(fmt.Sprintf(" { return \"type(%s)\" }", sig.receiver))
			buf.WriteString("\n")
		}

		buf.WriteString(line)
		buf.WriteString("\n")
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("%+v", err)
	}

	if err := os.WriteFile(filename, src, 0o644); err != nil {
		log.Fatalf("%+v", err)
	}
}

func getLines(data []byte) []string {
	var lines []string

	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func parseMethodSignature(line string) methodSignature {
	sig := methodSignature{}
	insideReceiver := false
	sb := strings.Builder{}

	for _, r := range line {
		if r == '(' {
			insideReceiver = true
			continue
		}

		if r == ')' {
			break
		}

		if insideReceiver {
			sb.WriteRune(r)
		}
	}

	parts := strings.Split(sb.String(), " ")

	sig.name = parts[0]
	sig.receiver = parts[1]

	return sig
}
