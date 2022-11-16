package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"net/http"
	"time"
)

var zlog, tracer = logging.RootLogger("beacon-proxy", "github.com/emiliocramer/lighthouse-geth-proxy/cmd/beacon-proxy")

func init() {
	logging.InstantiateLoggers()
}

var rootCmd = &cobra.Command{
	Use:   "beacon-proxy",
	Short: "Multi Reader-Node Proxy",
}

//func main() {
//	if err := rootCmd.Execute(); err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//}

func main() {
	Run("beacon-proxy", "Multi Reader-Node Proxy",
		CommandOptionFunc(func(parent *cobra.Command) {
			parent.AddCommand(ServeJSONRPCCommand)
		}),
		ConfigureViper("LIGHTHOUSE"),
		//ConfigureVersion(),
		PersistentFlags(
			func(flags *pflag.FlagSet) {
				flags.Duration("delay-before-start", 0, "[OPERATOR] Amount of time to wait before starting any internal processes, can be used to perform to maintenance on the pod before actually letting it starts")
				flags.String("metrics-listen-addr", "localhost:9102", "[OPERATOR] If non-empty, the process will listen on this address for Prometheus metrics request(s)")
				flags.String("pprof-listen-addr", "localhost:6060", "[OPERATOR] If non-empty, the process will listen on this address for pprof analysis (see https://golang.org/pkg/net/http/pprof/)")
			},
		),
		AfterAllHook(func(cmd *cobra.Command) {
			cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
				delay := viper.GetDuration("global-delay-before-start")
				if delay > 0 {
					zlog.Info("sleeping to respect delay before start setting", zap.Duration("delay", delay))
					time.Sleep(delay)
				}

				if v := viper.GetString("global-metrics-listen-addr"); v != "" {
					zlog.Info("starting prometheus metrics server", zap.String("listen_addr", v))
					// go dmetrics.Serve(v)
				}

				if v := viper.GetString("global-pprof-listen-addr"); v != "" {
					go func() {
						zlog.Info("starting pprof server", zap.String("listen_addr", v))
						err := http.ListenAndServe(v, nil)
						if err != nil {
							zlog.Debug("unable to start profiling server", zap.Error(err), zap.String("listen_addr", v))
						}
					}()
				}
			}
		}),
	)
}
