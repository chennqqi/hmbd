package main

import (
	"context"
	"flag"
	"time"

	"github.com/google/subcommands"
)

type webCmd struct {
	w      Web
	port   int
	fileto string
	zipto  string
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
	f.StringVar(&p.zipto, "timeout", "60s", "set scan timeout")
}

func (p *webCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	to, err := time.ParseDuration(p.zipto)
	if err != nil {
		to, _ = time.ParseDuration("60s")
	}
	p.w.fileto = to
	p.w.zipto = to
	p.w.Run(p.port)
	return subcommands.ExitSuccess
}
