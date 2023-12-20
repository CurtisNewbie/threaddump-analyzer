package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/curtisnewbie/threaddump-analyzer/analyzer"
)

var (
	StackDump  = flag.String("file", "", "path to stack dump")
	Details    = flag.Bool("details", false, "print all details")
	ReportFile = flag.Bool("report", false, "output report file including all the details")
)

func main() {
	fmt.Printf("\nthreaddump-analyzer %v, github.com/CurtisNewbie/threaddump-analyzer \n\n", analyzer.Version)
	flag.Parse()

	if StackDump == nil || *StackDump == "" {
		return
	}

	f := *StackDump
	content, err := analyzer.LoadStackFile(f)
	if err != nil {
		panic(err)
	}
	stack, err := analyzer.NewStack(content)
	if err != nil {
		panic(err)
	}

	opt := analyzer.StackOutputOption{
		Details: (Details != nil && *Details) || (ReportFile != nil && *ReportFile),
	}
	out := analyzer.StackOutput(stack, opt)

	if ReportFile == nil || !*ReportFile {
		fmt.Print(out)
		return
	}

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
