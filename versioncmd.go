package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/malice-plugins/go-plugin-utils/utils"
)

type versionCmd struct {
}

func (p *versionCmd) Name() string {
	return "version"
}

func (p *versionCmd) Synopsis() string {
	return `version `
}

func (p *versionCmd) Usage() string {
	return `version`
}

func (p *versionCmd) SetFlags(*flag.FlagSet) {
}

func (p *versionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ctx := context.TODO()
	r, err := utils.RunCommand(ctx, "hmb", "version")
	if err==nil{
		fmt.Println(r)
	} else {
		fmt.Println(r,err)
	}
	return subcommands.ExitSuccess
}
