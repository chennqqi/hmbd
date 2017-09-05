package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/malice-plugins/go-plugin-utils/utils"
)

type scanCmd struct {
}

func (p *scanCmd) Name() string {
	return "scan"
}

func (p *scanCmd) Synopsis() string {
	return "scan webshell in specific directory"
}

func (p *scanCmd) Usage() string {
	return "scan <targetdir>"
}

func (p *scanCmd) SetFlags(f *flag.FlagSet) {
}

func (p *scanCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	dirs := flag.Args()
	if len(dirs) == 0 {
		fmt.Println("target dir is must")
		return subcommands.ExitUsageError
	}
	var params []string
	params = append(params, "scan")
	params = append(params, dirs...)
	ctx := context.TODO()
	fmt.Println(utils.RunCommand(ctx, "hmb", params...))
	return subcommands.ExitSuccess
}
