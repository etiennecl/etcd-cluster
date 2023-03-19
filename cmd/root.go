package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/clinia/x/configx"
	"github.com/etiennemtl/etcd-mini-cluster/cmd/server"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "etcd-mini-cluster",
	}

	configx.RegisterConfigFlag(cmd.PersistentFlags(), []string{filepath.Join(userHomeDir(), "config.yml")})

	cmd.AddCommand(server.NewServeCmd())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happend once to the RotCmd.
func Execute() {
	c := NewRootCmd()

	if err := c.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
