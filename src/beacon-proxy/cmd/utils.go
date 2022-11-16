package cmd

import (
	"go.uber.org/zap"
	"os"
)

func maybeExitWithError(err error) error {
	if err == nil {
		zlog.Sync()
		os.Exit(0)
	}

	return exitWithError(err)
}

func exitWithError(err error) error {
	zlog.Error("app terminated unexpectedly", zap.Error(err))
	zlog.Sync()

	os.Exit(1)
	return nil
}
