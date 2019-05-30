package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

// Version is set with `go build -ldflags="-X github.com/puppetlabs/wash/cmd.version=${VERSION}"`
// as part of tagged builds. A local build might use `cmd.version=$(git describe --always)` instead.
var version = "unknown"

func versionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print wash version",
		RunE:  toRunE(versionMain),
	}
	return versionCmd
}

func versionMain(cmd *cobra.Command, args []string) exitCode {
	cmdutil.Println(version)
	return exitCode{0}
}
