package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streamingfast/cli"
	"github.com/streamingfast/dauth/authenticator"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/dmetering"
	"go.uber.org/zap"
	"time"
)

var ServeJSONRPCCommand = Command(serveJSONRPCE,
	"json-rpc",
	"Starts the JSON-RPC server",
	Description(`
		Opens up a JSON-RPC server that accepts 'eth_call' method request
	`),
	Flags(func(flags *pflag.FlagSet) {
		flags.String("listen-addr", ":8080", "The port that should be listened too for incoming JSON-RPC requests")
		flags.String("chain", "battlefield", FlagDescription(`
			Network name's transaction are going to be simulated for, this is used to determine the chain's config which
			includes actual activated EIPs and at which block, the chain ID and such other parameters that affects the
			EVM execution. We support most chain config pre-populated in Geth, as well as a 'battlefield' chain that fits
			out Ethereum Battlefield configuration.
		`))
		flags.Uint64("gas-cap", defaultGasCap, "Maximum amount of Gas that will ever be allowed for a call, if 'gas-limit' is higher than 'gas-cap', the server will reduce it back to the maximum which is 'gas-cap'")
		flags.Duration("timeout", defaultExecuteTimeout, "Maximum amount of time allow for a single 'eth_call' execution before it's killed")
		flags.String("state-provider-dsn", "localhost:9000", "State provider DSN used to instantiate executor")
	}),
)

func serveJSONRPCE(cmd *cobra.Command, _ []string) error {
	chain := viper.GetString("serve-json-rpc-chain")
	gasCap := viper.GetUint64("serve-json-rpc-gas-cap")
	listenAddr := viper.GetString("serve-json-rpc-listen-addr")
	timeout := viper.GetDuration("serve-json-rpc-timeout")
	stateProviderDSN := viper.GetString("serve-json-rpc-state-provider-dsn")

	zlog.Info("starting server",
		zap.String("chain", chain),
		zap.Uint64("gas_cap", gasCap),
		zap.String("listen_addr", listenAddr),
		zap.String("state_provider_dsn", stateProviderDSN),
	)

	authenticator, err := authenticator.New(viper.GetString("global-common-auth-plugin"))
	if err != nil {
		return fmt.Errorf("unable to initialize dauth: %w", err)
	}

	metering, err := dmetering.New(viper.GetString("global-common-metering-plugin"))
	if err != nil {
		return fmt.Errorf("unable to initialize dmetering: %w", err)
	}
	dmetering.SetDefaultMeter(metering)

	chainConfig := evmexecutor.NetworkNameToChainConfig(chain)
	cli.Ensure(chainConfig != nil, "Unsupported chain %q", chain)

	executor, err := evmexecutor.NewCallExecutor(cmd.Context(), chainConfig, gasCap, stateProviderDSN, timeout)
	if err != nil {
		return fmt.Errorf("new executor: %w", err)
	}

	server, err := jsonrpc.NewServer(
		listenAddr,
		func() bool { return true },
		[]services.ServiceHandler{
			services.NewEthService(executor),
			services.NewNetService(chainConfig.NetworkID),
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
