package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/google/subcommands"
)

type webCmd struct {
	w        Web
	port     int
	fileto   string
	zipto    string
	callback string
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
	f.StringVar(&p.callback, "callback", "", "set callback addr")
}

func (p *webCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	to, err := time.ParseDuration(p.zipto)
	if err != nil {
		to, _ = time.ParseDuration("60s")
	}
	if p.callback == "" {
		p.callback = os.Getenv("HMBD_CALLBACK")
	}

	p.w.fileto = to
	p.w.zipto = to
	p.w.callback = p.callback
	p.w.Run(p.port)
	return subcommands.ExitSuccess
}
