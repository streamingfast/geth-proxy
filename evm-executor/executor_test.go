package evmexecutor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/remeh/sizedwaitgroup"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go/rpc"
	"github.com/stretchr/testify/require"
)

func TestEthereumGoerliTests(t *testing.T) {
	stateProviderDSN := os.Getenv("EVM_EXECUTOR_GOERLI_STATE_PROVIDER_DSN")
	if stateProviderDSN == "" {
		t.Skip("the environment variable EVM_EXECUTOR_GOERLI_STATE_PROVIDER_DSN must be set for this tests to run properly")
	}

	ctx := context.Background()

	executor, err := NewCallExecutor(ctx, NetworkNameToChainConfig("goerli"), 550_000_000, stateProviderDSN, time.Second*5)
	require.NoError(t, err)

	data, gasUsed, err := executor.ExecuteCall(ctx, rpc.CallParams{}, bstream.NewBlockRef("", 1847577))

	require.NoError(t, err)
	require.Equal(t, "", hex.EncodeToString(data))
	require.Equal(t, 0, gasUsed)
}

func TestEthereumStateTests(t *testing.T) {
	ethereumTestsDir := os.Getenv("GO_ETHEREUM_TESTS_DIR")
	if ethereumTestsDir == "" {
		t.Skip("the environment variable GO_ETHEREUM_TESTS_DIR must be set for this tests to run properly")
	}

	stateTests := findEthereumTests(t, ethereumTestsDir, "^GeneralStateTests")

	for from, stateTestsMap := range stateTests {
		runStateTest(t, from, stateTestsMap)
	}
}

func findEthereumTests(t *testing.T, dir string, match string) (out map[string]map[string]StateTest) {
	t.Helper()
	out = map[string]map[string]StateTest{}

	waitGroup := sizedwaitgroup.New(8)
	lock := sync.Mutex{}
	matchRegex := regexp.MustCompile(match)

	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relativePath := strings.TrimLeft(strings.TrimPrefix(path, dir), "/")

		if d.IsDir() {
			if relativePath == "src" || relativePath == "docs" || relativePath == "ansible" {
				return filepath.SkipDir
			}

			return nil
		}

		if !matchRegex.MatchString(relativePath) {
			return nil
		}

		if !strings.HasSuffix(relativePath, ".json") {
			return nil
		}

		waitGroup.Add()
		go func(fullPath string, relativePath string) {
			defer waitGroup.Done()

			stateTestsMap, err := readStateTestFromFile(fullPath)
			require.NoError(t, err)

			lock.Lock()
			out[relativePath] = stateTestsMap
			lock.Unlock()
		}(path, relativePath)

		waitGroup.Wait()

		return nil
	})

	return
}

func readStateTestFromFile(file string) (out map[string]StateTest, err error) {
	src, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var tests map[string]StateTest
	if err := json.Unmarshal(src, &tests); err != nil {
		return nil, fmt.Errorf("decode %q: %w", file, err)
	}

	return tests, nil
}

func runStateTest(t *testing.T, from string, tests map[string]StateTest) {
	require.True(t, len(tests) > 0, "expected at least one state test from %q, got 0", from)

	for name, test := range tests {
		require.True(t, len(test.Subtests()) > 0, "expected at least one state sub test on %q from %q, got 0", name, from)

		for _, subtest := range test.Subtests() {
			subtest := subtest
			key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

			t.Run(key, func(t *testing.T) {
				withTrace(t, test.gasLimit(subtest), func(vmconfig vm.Config) error {
					_, _, err := test.Run(subtest, vmconfig, false)
					if err != nil && len(test.json.Post[subtest.Fork][subtest.Index].ExpectException) > 0 {
						// Ignore expected errors (TODO MariusVanDerWijden check error string)
						return nil
					}

					// This was checkFailure before, goal was to set "globally" which test would fail
					// and with which message. Instead of having to fill the table with all possible
					// tests, we configure in a registry which one should fail and how and we check it
					// here based on the test name.
					return nil
				})
			})
		}
	}
}

// Transactions with gasLimit above this value will not get a VM trace on failure.
const traceErrorLimit = 400000

func withTrace(t *testing.T, gasLimit uint64, test func(vm.Config) error) {
	// Use config from command line arguments.
	config := vm.Config{}
	err := test(config)
	if err == nil {
		return
	}

	// Test failed, re-run with tracing enabled.
	t.Error(err)
	if gasLimit > traceErrorLimit {
		t.Log("gas limit too high for EVM trace")
		return
	}
	buf := new(bytes.Buffer)
	w := bufio.NewWriter(buf)
	tracer := logger.NewJSONLogger(&logger.Config{}, w)
	config.Debug, config.Tracer = true, tracer
	err2 := test(config)
	if !reflect.DeepEqual(err, err2) {
		t.Errorf("different error for second run: %v", err2)
	}
	w.Flush()
	if buf.Len() == 0 {
		t.Log("no EVM operation logs generated")
	} else {
		t.Log("EVM operation log:\n" + buf.String())
	}
}

func unifiedDiff(t *testing.T, cnt1, cnt2 []byte) string {
	file1 := "/tmp/gotests-evm-executor-linediff-1"
	file2 := "/tmp/gotests-evm-executor-linediff-2"
	err := ioutil.WriteFile(file1, cnt1, 0600)
	require.NoError(t, err)

	err = ioutil.WriteFile(file2, cnt2, 0600)
	require.NoError(t, err)

	cmd := exec.Command("diff", "-u", file1, file2)
	buffer, _ := cmd.Output()

	out := string(buffer)
	if tracer.Enabled() {
		out += "\nExpected:\n" + string(cnt1) + "\n\nActual:\n" + string(cnt2)
	}

	return out
}
