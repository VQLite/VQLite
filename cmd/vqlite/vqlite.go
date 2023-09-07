package vqlite

import (
	"flag"
	"fmt"
	"os"
)

type command interface {
	execute(args []string, flags *flag.FlagSet)
}

type defaultCommand struct{}

func (c *defaultCommand) execute(args []string, flags *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "unknown command : %s\n", args[1])
	fmt.Fprintln(os.Stdout, usageLine)
}

func RunVQlite(args []string) {

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, usageLine)
		return
	}

	cmd := args[1]
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usageLine)
	}

	var c command
	switch cmd {
	case RunCmd:
		c = &run{}
	case TrainCmd:
		c = &train{}
	default:
		c = &defaultCommand{}
	}

	c.execute(args, flags)

	//if len(os.Args) > 2 {
	//	if os.Args[1] == "train" {
	//		ret := core.TrainSegmentByCmd()
	//		os.Exit(ret)
	//	}
	//}
	//
	//host := config.GlobalConfig.ServiceConfig.Host
	//port := config.GlobalConfig.ServiceConfig.Port
	//addr := fmt.Sprintf("%s:%d", host, port)
	//
	//r := routes.InitRouter()
	//r.Run(addr)
}
