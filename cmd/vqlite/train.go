package vqlite

import (
	"flag"
	"fmt"
	"os"
	"vqlite/core"
)

const (
	TrainCmd = "train"
)

type train struct {
	segmentWorkDir string
	numThreads     int
}

func (t *train) execute(args []string, flags *flag.FlagSet) {

	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, usageLine)
		return
	}
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usageLine)
	}

	t.formatFlags(args, flags)

	ret := core.TrainSegmentByCmd(t.segmentWorkDir, t.numThreads)
	os.Exit(ret)

}

func (t *train) formatFlags(args []string, flags *flag.FlagSet) {
	flags.StringVar(&t.segmentWorkDir, "segmentWorkDir", "", "segment work dir")
	flags.IntVar(&t.numThreads, "numThreads", 0, "num threads")
	if err := flags.Parse(args[2:]); err != nil {
		os.Exit(-1)
	}
}
