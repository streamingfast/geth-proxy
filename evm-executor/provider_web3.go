package evmexecutor

import (
	"context"
	"fmt"
	"math/big"
	"net/url"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

const JSONRPCScheme = "json-rpc"
const StateDBScheme = "statedb"

var EmptyHash = [common.HashLength]byte{}

type JSONRPCStateProvider struct {
	client *ethclient.Client
}

func NewJSONRPCStateProvider(client *ethclient.Client) *JSONRPCStateProvider {
	return &JSONRPCStateProvider{
		client: client,
	}
}

func NewJSONRPCStateProviderFromDSN(ctx context.Context, dsn string) (*JSONRPCStateProvider, error) {
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	if dsnURL.Scheme != JSONRPCScheme {
		return nil, fmt.Errorf("invalid scheme %q, accepting only %q", dsnURL.Scheme, JSONRPCScheme)
	}

	host := dsnURL.Host + "/" + dsnURL.Path
	scheme := "https://"
	if secureParam := dsnURL.Query().Get("secure"); secureParam == "false" {
		scheme = "http://"
	}

	client, err := rpc.DialContext(ctx, scheme+host)
	if err != nil {
		return nil, fmt.Errorf("rpc dial: %w", err)
	}

	return NewJSONRPCStateProvider(ethclient.NewClient(client)), nil
}

func (p *JSONRPCStateProvider) FetchAccount(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error) {
	logging.Logger(ctx, zlog).Debug("fetching account", zap.Stringer("addr", addr), zap.Stringer("block", atBlock))
	web3BlockNum := web3BlockNum(atBlock)

	balance, err := p.client.BalanceAt(ctx, addr, web3BlockNum)
	if err != nil {
		return nil, normalizeRpcError(err)
	}

	nonce, err := p.client.NonceAt(ctx, addr, web3BlockNum)
	if err != nil {
		return nil, normalizeRpcError(err)
	}

	code, err := p.client.CodeAt(ctx, addr, web3BlockNum)
	if err != nil {
		return nil, normalizeRpcError(err)
	}

	return NewAccount(nonce, balance, code), nil
}

func (p *JSONRPCStateProvider) FetchBlockHeaderByHash(ctx context.Context, hash eth.Hash) (*types.Header, error) {
	logging.Logger(ctx, zlog).Debug("fetching block header by hash", zap.Stringer("block_hash", hash))

	block, err := p.client.BlockByHash(ctx, common.BytesToHash(hash))
	if err != nil {
		return nil, err
	}

	return block.Header(), nil
}

func (p *JSONRPCStateProvider) FetchBlockHeaderByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	logging.Logger(ctx, zlog).Debug("fetching block header by number", zap.Uint64("number", number))

	block, err := p.client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return nil, err
	}

	return block.Header(), nil
}

func (p *JSONRPCStateProvider) FetchLatestBlockHeader(ctx context.Context) (*types.Header, error) {
	logging.Logger(ctx, zlog).Debug("fetching latest block header")

	latestBlockNumber, err := p.client.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	block, err := p.client.BlockByNumber(ctx, new(big.Int).SetUint64(latestBlockNumber))
	if err != nil {
		return nil, err
	}

	return block.Header(), nil
}

func (p *JSONRPCStateProvider) FetchStorage(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (common.Hash, error) {
	logging.Logger(ctx, zlog).Debug("fetching storage value", zap.Stringer("addr", addr), zap.Stringer("key", key), zap.Stringer("block", atBlock))

	value, err := p.client.StorageAt(ctx, addr, key, web3BlockNum(atBlock))
	if err != nil {
		return EmptyHash, normalizeRpcError(err)
	}

	return common.BytesToHash(value), nil
}

func web3BlockNum(atBlock bstream.BlockRef) *big.Int {
	if isLatestBlockRef(atBlock) {
		return nil
	}

	return big.NewInt(int64(atBlock.Num()))
}

func normalizeRpcError(err error) error {
	if err == rpc.ErrNoResult {
		return ErrNotFound
	}

	return err
}
