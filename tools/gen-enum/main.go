package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var print bool

	options := GeneratorOptions{
		Args: os.Args[1:],
	}

	flag := flag.NewFlagSet("gen-enum", flag.ContinueOnError)

	flag.StringVar(&options.Type, "type", "", "type name")
	flag.BoolVar(&options.GenerateFlag, "generate-flag", false, "generate methods for flag interface")
	flag.StringVar(&options.Output, "output", "", "output file name; default srcdir/<filename_with_type>_enum.go")
	flag.StringVar(&options.BuildTags, "tags", "", "comma-separated list of build tags to apply")
	flag.BoolVar(&print, "print", false, "print the generated code to stdout")

	if err := flag.Parse(os.Args[1:]); err != nil {
		log.Fatalf("%+v", err)
	}

	if len(options.Type) == 0 {
		log.Printf("-type is required")
		os.Exit(1)
	}

	generator := NewGenerator(options)
	src, err := generator.Run()
	if err != nil {
		if print {
			fmt.Println(string(src))
		}

		log.Fatalf("%+v", err)
	}

	if print {
		fmt.Println(string(src))
	}
}
