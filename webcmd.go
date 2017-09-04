package main

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

type webCmd struct {
	w    Web
	port int
}

func (p *webCmd) Name() string {
	return "web"
}

func (p *webCmd) Synopsis() string {
	return "web"
}

func (p *webCmd) Usage() string {
	return "web -p port"
}

func (p *webCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&p.port, "p", 8080, "set port")
}

func (p *webCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	p.w.Run(p.port)
	return subcommands.ExitSuccess
}
