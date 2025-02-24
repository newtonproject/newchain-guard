package cli

import (
	"github.com/spf13/cobra"
)

func (cli *CLI) buildRootCmd() {
	if cli.rootCmd != nil {
		cli.rootCmd.ResetFlags()
		cli.rootCmd.ResetCommands()
	}

	rootCmd := &cobra.Command{
		Use:              cli.name,
		Short:            cli.name + " is a commandline server for the NewChain Proxy",
		Run:              cli.help,
		PersistentPreRun: cli.setup,
	}
	cli.rootCmd = rootCmd

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cli.configpath, "config", "c", defaultConfigFile, "The `path` to config file")
	rootCmd.PersistentFlags().StringP("rpcURL", "i", defaultRPCURL, "Geth json rpc or ipc `url`")

	// Basic commands
	rootCmd.AddCommand(cli.buildVersionCmd()) // version
	rootCmd.AddCommand(cli.buildInitCmd())    // init

	// server
	rootCmd.AddCommand(cli.buildServerCmd()) // server
}
