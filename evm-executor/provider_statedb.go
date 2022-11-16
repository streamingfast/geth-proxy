package evmexecutor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"sync"

	pbstatedb "github.com/emiliocramer/lighthouse-geth-proxy/pb/sf/ethereum/statedb/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dgrpc"
	"github.com/streamingfast/eth-go"
	pbcodec "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"github.com/streamingfast/logging"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ StateProvider = (*StateDBProvider)(nil)

type StateDBProvider struct {
	client pbstatedb.StateClient
}

func NewStateDBProvider(client pbstatedb.StateClient) *StateDBProvider {
	return &StateDBProvider{
		client: client,
	}
}

func NewStateDBProviderFromDSN(ctx context.Context, dsn string) (*StateDBProvider, error) {
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	if dsnURL.Scheme != StateDBScheme {
		return nil, fmt.Errorf("invalid scheme %q, accepting only %q", dsnURL.Scheme, StateDBScheme)
	}

	endpoint := dsnURL.Host
	if dsnURL.Path != "" {
		endpoint += "/" + dsnURL.Path
	}

	secureParam := dsnURL.Query().Get("secure") == "true"
	insecureParam := dsnURL.Query().Get("insecure") == "true"
	shouldUseTLS := secureParam || insecureParam

	var conn *grpc.ClientConn
	if shouldUseTLS {
		if insecureParam {
			return nil, fmt.Errorf("insecure=true is not supported yet")
		}

		logging.Logger(ctx, zlog).Debug("connecting to a TLS StateDB endpoint", zap.String("endpoint", endpoint))
		conn, err = dgrpc.NewExternalClient(endpoint)
	} else {
		logging.Logger(ctx, zlog).Debug("connecting to a plain-text StateDB endpoint", zap.String("endpoint", endpoint))
		conn, err = dgrpc.NewInternalClient(endpoint)
	}

	if err != nil {
		return nil, fmt.Errorf("grpc conn: %w", err)
	}

	return NewStateDBProvider(pbstatedb.NewStateClient(conn)), nil
}

func (p *StateDBProvider) FetchAccount(ctx context.Context, addr common.Address, atBlock bstream.BlockRef) (*Account, error) {
	logging.Logger(ctx, zlog).Debug("fetching account", zap.Stringer("addr", addr), zap.Stringer("block", atBlock))

	account := &Account{}
	waitGroup := &sync.WaitGroup{}
	lock := sync.Mutex{}
	var activeErr error

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		response, err := p.client.GetBalance(ctx, &pbstatedb.GetBalanceRequest{Address: addr[:], BlockRef: blockRefToProto(atBlock)})

		lock.Lock()
		if err != nil {
			if status.Code(err) == codes.NotFound {
				account.Balance = common.Big0
			} else {
				activeErr = multierr.Append(activeErr, fmt.Errorf("get balance: %w", err))
			}
		} else {
			account.Balance = new(big.Int).SetBytes(response.Balance)
		}
		lock.Unlock()
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		response, err := p.client.GetNonce(ctx, &pbstatedb.GetNonceRequest{Address: addr[:], BlockRef: blockRefToProto(atBlock)})

		lock.Lock()
		if err != nil {
			if status.Code(err) != codes.NotFound {
				activeErr = multierr.Append(activeErr, fmt.Errorf("get nonce: %w", err))
			}
		} else {
			account.Nonce = response.Nonce
		}
		lock.Unlock()
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		// FIXME: We will need to change the interface to not always fetch the code, it's probably not required always
		//        but that's something unclear it might be always required to have it set to the user's account. Or, we
		//        add a speciallized `GetAccount` on StateDB interface to avoid doing N api calls to retriev all the data.
		response, err := p.client.GetCode(ctx, &pbstatedb.GetCodeRequest{Address: addr[:], BlockRef: blockRefToProto(atBlock)})

		lock.Lock()
		if err != nil {
			if status.Code(err) == codes.NotFound {
				account.Code = []byte{}
				account.CodeHash = nil
			} else {
				activeErr = multierr.Append(activeErr, fmt.Errorf("get code: %w", err))
			}
		} else {
			account.Code = response.Payload
			account.CodeHash = computeCodeHash(response.Payload)
		}
		lock.Unlock()
	}()

	waitGroup.Wait()
	if activeErr != nil {
		return nil, activeErr
	}

	return account, nil
}

func (p *StateDBProvider) FetchBlockHeaderByHash(ctx context.Context, hash eth.Hash) (*types.Header, error) {
	logging.Logger(ctx, zlog).Debug("fetching block header by hash", zap.Stringer("block_hash", hash))

	response, err := p.client.GetBlockHeaderByHash(ctx, &pbstatedb.GetBlockHeaderByHashRequest{Hash: hash})
	if err != nil {
		return nil, fmt.Errorf("get block header by hash: %w", err)
	}

	return codecHeaderToGethHeader(response.Header), nil
}

func (p *StateDBProvider) FetchBlockHeaderByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	logging.Logger(ctx, zlog).Debug("fetching block header by number", zap.Uint64("block_number", number))

	response, err := p.client.GetBlockHeaderByNumber(ctx, &pbstatedb.GetBlockHeaderByNumberRequest{Number: number})
	if err != nil {
		return nil, fmt.Errorf("get block header by number: %w", err)
	}

	return codecHeaderToGethHeader(response.Header), nil
}

func (p *StateDBProvider) FetchLatestBlockHeader(ctx context.Context) (*types.Header, error) {
	response, err := p.client.GetLatestBlockHeader(ctx, &pbstatedb.GetLatestBlockHeaderRequest{})
	if err != nil {
		return nil, fmt.Errorf("get latest block header: %w", err)
	}

	return codecHeaderToGethHeader(response.Header), nil
}

func (p *StateDBProvider) FetchStorage(ctx context.Context, addr common.Address, key common.Hash, atBlock bstream.BlockRef) (out common.Hash, err error) {
	logging.Logger(ctx, zlog).Debug("fetching storage value", zap.Stringer("addr", addr), zap.Stringer("key", key), zap.Stringer("block", atBlock))

	response, err := p.client.GetState(ctx, &pbstatedb.GetStateRequest{Address: addr[:], Key: key[:], BlockRef: blockRefToProto(atBlock)})
	if err != nil {
		var status (interface{ GRPCStatus() *status.Status })
		if errors.As(err, &status) && status.GRPCStatus().Code() == codes.NotFound {
			return out, ErrNotFound
		}

		return out, fmt.Errorf("fetch storage: %w", err)
	}

	return common.BytesToHash(response.Data), nil
}

func codecHeaderToGethHeader(header *pbcodec.BlockHeader) *types.Header {
	// The base fee must be set only if it's set but `Native` returns 0 in
	// those case, so check first if it's non-nil before converting it.
	var baseFee *big.Int
	if header.BaseFeePerGas != nil && header.BaseFeePerGas.Bytes != nil {
		baseFee = header.BaseFeePerGas.Native()
	}

	return &types.Header{
		ParentHash:  common.BytesToHash(header.ParentHash),
		UncleHash:   common.BytesToHash(header.UncleHash),
		Coinbase:    common.BytesToAddress(header.Coinbase),
		Root:        common.BytesToHash(header.StateRoot),
		TxHash:      common.BytesToHash(header.TransactionsRoot),
		ReceiptHash: common.BytesToHash(header.ReceiptRoot),
		Bloom:       types.BytesToBloom(header.LogsBloom),
		Difficulty:  header.Difficulty.Native(),
		Number:      new(big.Int).SetUint64(header.Number),
		GasLimit:    header.GasLimit,
		GasUsed:     header.GasUsed,
		Time:        uint64(header.Timestamp.Seconds),
		Extra:       header.ExtraData,
		MixDigest:   common.BytesToHash(header.MixHash),
		Nonce:       types.EncodeNonce(header.Nonce),
		BaseFee:     baseFee,
	}
}

func blockRefToProto(ref bstream.BlockRef) *pbbstream.BlockRef {
	return &pbbstream.BlockRef{
		Id:  ref.ID(),
		Num: ref.Num(),
	}
}
