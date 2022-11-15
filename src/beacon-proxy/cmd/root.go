package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var rootCmd = &cobra.Command{
	Use:   "beacon-proxy",
	Short: "Multi Reader-Node Proxy",
}

var defaultGasCap = uint64(550_000_000)
var defaultExecuteTimeout = time.Second * 300

func Main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
