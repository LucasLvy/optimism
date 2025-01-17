package contracts

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
)

const (
	methodGameCount   = "gameCount"
	methodGameAtIndex = "gameAtIndex"
	methodGameImpls   = "gameImpls"
)

type DisputeGameFactoryContract struct {
	multiCaller *batching.MultiCaller
	contract    *batching.BoundContract
}

func NewDisputeGameFactoryContract(addr common.Address, caller *batching.MultiCaller) (*DisputeGameFactoryContract, error) {
	factoryAbi, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to load dispute game factory ABI: %w", err)
	}
	return &DisputeGameFactoryContract{
		multiCaller: caller,
		contract:    batching.NewBoundContract(factoryAbi, addr),
	}, nil
}

func (f *DisputeGameFactoryContract) GetGameCount(ctx context.Context, blockHash common.Hash) (uint64, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockByHash(blockHash), f.contract.Call(methodGameCount))
	if err != nil {
		return 0, fmt.Errorf("failed to load game count: %w", err)
	}
	return result.GetBigInt(0).Uint64(), nil
}

func (f *DisputeGameFactoryContract) GetGame(ctx context.Context, idx uint64, blockHash common.Hash) (types.GameMetadata, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockByHash(blockHash), f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(idx)))
	if err != nil {
		return types.GameMetadata{}, fmt.Errorf("failed to load game %v: %w", idx, err)
	}
	return f.decodeGame(result), nil
}

func (f *DisputeGameFactoryContract) GetGameImpl(ctx context.Context, gameType uint8) (common.Address, error) {
	result, err := f.multiCaller.SingleCall(ctx, batching.BlockLatest, f.contract.Call(methodGameImpls, gameType))
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to load game impl for type %v: %w", gameType, err)
	}
	return result.GetAddress(0), nil
}

func (f *DisputeGameFactoryContract) GetAllGames(ctx context.Context, blockHash common.Hash) ([]types.GameMetadata, error) {
	count, err := f.GetGameCount(ctx, blockHash)
	if err != nil {
		return nil, err
	}

	calls := make([]*batching.ContractCall, count)
	for i := uint64(0); i < count; i++ {
		calls[i] = f.contract.Call(methodGameAtIndex, new(big.Int).SetUint64(i))
	}

	results, err := f.multiCaller.Call(ctx, batching.BlockByHash(blockHash), calls...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch games: %w", err)
	}

	var games []types.GameMetadata
	for _, result := range results {
		games = append(games, f.decodeGame(result))
	}
	return games, nil
}

func (f *DisputeGameFactoryContract) decodeGame(result *batching.CallResult) types.GameMetadata {
	gameType := result.GetUint8(0)
	timestamp := result.GetUint64(1)
	proxy := result.GetAddress(2)
	return types.GameMetadata{
		GameType:  gameType,
		Timestamp: timestamp,
		Proxy:     proxy,
	}
}
