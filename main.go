package main

import (
	"fmt"
	"os"

	"github.com/curtisnewbie/threaddump-analyzer/analyzer"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		return
	}
	args = args[1:]
	for _, f := range args {
		content, err := analyzer.LoadStackFile(f)
		if err != nil {
			panic(err)
		}
		stack, err := analyzer.NewStack(content)
		if err != nil {
			panic(err)
		}
		out := analyzer.StackOutput(stack)
		fn := f
		fnr := []rune(f)
		for i := len(fnr) - 1; i >= 0; i-- {
			if fnr[i] == '.' {
				fn = string(fnr[:i])
				break
			}
		}
		report := fn + "_report.txt"
		if err := os.WriteFile(report, []byte(out), os.ModePerm); err != nil {
			panic(err)
		}
		fmt.Printf("Created report '%v' for dump '%v'\n", report, f)
	}
}
