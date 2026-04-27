package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lusterpass",
	Short: "Manage app secrets via Bitwarden with encrypted local caching",
}

func SetVersion(v string) {
	rootCmd.Version = v
}

var configFile string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", ".lusterpass.yaml", "Path to config file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
