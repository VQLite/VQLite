package vqlite

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"vqlite/config"
	routes "vqlite/routers"
)

const (
	RunCmd = "run"
)

type run struct {
	host string
	port string
}

func (c *run) execute(args []string, flags *flag.FlagSet) {

	host := config.GlobalConfig.ServiceConfig.Host
	port := config.GlobalConfig.ServiceConfig.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	if len(args) > 2 {
		c.formatFlags(args, flags)
	}
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usageLine)
	}

	// make go ignore SIGPIPE when all cgo threads set mask of SIGPIPE
	signal.Ignore(syscall.SIGPIPE)

	r := routes.InitRouter()
	r.Run(addr)
}

func (c *run) formatFlags(args []string, flags *flag.FlagSet) {

	flags.StringVar(&c.host, "host", "127.0.0.1", "VQLite service address")
	flags.StringVar(&c.port, "port", "8880", "VQLite service port")
	if err := flags.Parse(args[2:]); err != nil {
		os.Exit(-1)
	}
}
