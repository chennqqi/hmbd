package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/google/subcommands"
	"github.com/malice-plugins/go-plugin-utils/utils"
)

var versionExp = regexp.MustCompile(`(?m)(\d+\.\d+\.\d+)`)

type updateCmd struct {
	c subcommands.Command
}

func (p *updateCmd) Name() string {
	return "update"
}

func (p *updateCmd) Synopsis() string {
	return "update"
}

func (p *updateCmd) Usage() string {
	return "update"
}

func (p *updateCmd) SetFlags(*flag.FlagSet) {
}

func (p *updateCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	{
		ctx := context.TODO()
		r, err := utils.RunCommand(ctx, "hmb", "update")
		fmt.Println(r)
		if err != nil {
			return subcommands.ExitFailure
		}
		//TODO:: parse is updated
	}

	{
		ctx := context.TODO()
		r, err := utils.RunCommand(ctx, "hmb", "version")
		if err == nil {
			v := versionExp.FindAllStringSubmatch(r, 1)
			if len(v) > 0 {
				ioutil.WriteFile("/opt/hmb/VERSION", []byte(v[0][1]), 0644)
			}
		}
	}
	return subcommands.ExitSuccess
}
