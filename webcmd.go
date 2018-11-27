package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chennqqi/goutils/closeevent"
	unet "github.com/chennqqi/goutils/net"
	utime "github.com/chennqqi/goutils/time"
	"github.com/google/subcommands"
)

type webCmd struct {
	w        *Web
	port     int
	fileto   string
	zipto    string
	callback string
	datadir  string
	indexdir string
	dns      string
	batch    int
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
	f.StringVar(&p.zipto, "timeout", "60s", "set scan dir timeout")
	f.StringVar(&p.fileto, "fileto", "20s", "set scan file timeout")
	f.StringVar(&p.callback, "callback", "", "set callback addr")
	f.StringVar(&p.datadir, "data", "/dev/shm", "set data dir")
	f.StringVar(&p.indexdir, "index", "/dev/shm/.persist", "set index dir")
	f.StringVar(&p.dns, "nameserver", "", "set ns server, can be list split by coma")
	f.IntVar(&p.batch, "batch", 1, "set batch route")
}

func (p *webCmd) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	zipTo, err := time.ParseDuration(p.zipto)
	if err != nil {
		zipTo, _ = time.ParseDuration("60s")
	}
	fileTo, err := time.ParseDuration(p.fileto)
	if err != nil {
		fileTo, _ = time.ParseDuration("20s")
	}

	if p.callback == "" {
		p.callback = os.Getenv("HMBD_CALLBACK")
	}

	if p.dns != "" {
		dns := strings.Split(p.dns, ",")
		if len(dns) > 0 {
			net.DefaultResolver = unet.NewResolver(dns)
		}
	} else {
		dns := os.Getenv("HMBD_DNS")
		dnslist := strings.Split(p.dns, ",")
		if dns != "" && len(dnslist) > 0 {
			net.DefaultResolver = unet.NewResolver(dnslist)
		}
	}

	w, err := NewWeb(p.datadir, p.indexdir, p.batch)
	if err != nil {
		fmt.Println("new web error:", err)
		return subcommands.ExitFailure
	}
	p.w = w
	p.w.fileto = utime.Duration(zipTo)
	p.w.zipto = utime.Duration(fileTo)
	p.w.callback = p.callback

	ctx, cancel := context.WithCancel(context.Background())
	go w.Run(p.port, ctx)

	closeevent.Wait(func(s os.Signal) {
		defer cancel()
		ctx := context.Background()
		w.Shutdown(ctx)
	}, os.Interrupt)

	return subcommands.ExitSuccess
}
