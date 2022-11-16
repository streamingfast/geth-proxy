package cmd

import (
	"fmt"
	"time"

	jsonrpc "github.com/emiliocramer/lighthouse-geth-proxy/json-rpc"
	"github.com/emiliocramer/lighthouse-geth-proxy/json-rpc/services"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dauth/authenticator"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dmetering"
)

func init() {
	rootCmd.AddCommand(ServeJSONRPCCommand)

	ServeJSONRPCCommand.PersistentFlags().String("listen-addr", ":8080", "The port that should be listened too for incoming JSON-RPC requests")
}

var ServeJSONRPCCommand = &cobra.Command{
	Use:   "listen",
	Short: "Starts the JSON-RPC server",
	Long:  "Opens up a JSON-RPC server that accepts 'eth_call' method request",
	RunE:  serveJSONRPCE,
}

func serveJSONRPCE(cmd *cobra.Command, args []string) error {
	listenAddr := viper.GetString("serve-json-rpc-listen-addr")
	
	zlog.Info("starting server")

	authenticator, err := authenticator.New(viper.GetString("global-common-auth-plugin"))
	if err != nil {
		return fmt.Errorf("unable to initialize dauth: %w", err)
	}

	metering, err := dmetering.New(viper.GetString("global-common-metering-plugin"))
	if err != nil {
		return fmt.Errorf("unable to initialize dmetering: %w", err)
	}
	dmetering.SetDefaultMeter(metering)

	server, err := jsonrpc.NewServer(
		listenAddr,
		func() bool { return true },
		[]services.ServiceHandler{
			services.NewEngineService(),
		},
		authenticator,
		metering,
	)

	if err != nil {
		return fmt.Errorf("creating json rpc server: %w", err)
	}

	go server.Serve()

	zlog.Info("waiting for server to terminate")

	select {
	case <-derr.SetupSignalHandler(0 * time.Second):
		server.Shutdown(nil)
		zlog.Info("signal handler terminated app, waiting for it to complete")
		<-server.Terminated()
	case <-server.Terminated():
		return maybeExitWithError(server.Err())
	}

	return nil
}
