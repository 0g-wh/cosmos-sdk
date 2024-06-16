package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GetBondedPool returns the bonded tokens pool's module account
func (k Keeper) GetBondedPool(ctx sdk.Context) (bondedPool authtypes.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.BondedPoolName)
}

// GetNotBondedPool returns the not bonded tokens pool's module account
func (k Keeper) GetNotBondedPool(ctx sdk.Context) (notBondedPool authtypes.ModuleAccountI) {
	return k.authKeeper.GetModuleAccount(ctx, types.NotBondedPoolName)
}

// bondedTokensToNotBonded transfers coins from the bonded to the not bonded pool within staking
func (k Keeper) bondedTokensToNotBonded(ctx sdk.Context, tokens math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), tokens))
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.BondedPoolName, types.NotBondedPoolName, coins); err != nil {
		panic(err)
	}
}

// notBondedTokensToBonded transfers coins from the not bonded to the bonded pool within staking
func (k Keeper) notBondedTokensToBonded(ctx sdk.Context, tokens math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), tokens))
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.NotBondedPoolName, types.BondedPoolName, coins); err != nil {
		panic(err)
	}
}

// burnBondedTokens removes coins from the bonded pool module account
func (k Keeper) burnBondedTokens(ctx sdk.Context, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), amt))

	return k.bankKeeper.BurnCoins(ctx, types.BondedPoolName, coins)
}

// burnNotBondedTokens removes coins from the not bonded pool module account
func (k Keeper) burnNotBondedTokens(ctx sdk.Context, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), amt))

	return k.bankKeeper.BurnCoins(ctx, types.NotBondedPoolName, coins)
}

// TotalBondedTokens total staking tokens supply which is bonded
func (k Keeper) TotalBondedTokens(ctx sdk.Context) math.Int {
	bondedPool := k.GetBondedPool(ctx)
	return k.bankKeeper.GetBalance(ctx, bondedPool.GetAddress(), k.BondDenom(ctx)).Amount
}

// StakingTokenSupply staking tokens from the total supply
func (k Keeper) StakingTokenSupply(ctx sdk.Context) math.Int {
	return k.bankKeeper.GetSupply(ctx, k.BondDenom(ctx)).Amount
}

func (k Keeper) StakingTokenCirculatingSupply(ctx sdk.Context) math.Int {
	totalSupply := k.StakingTokenSupply(ctx)

	totalLocked := math.ZeroInt()
	k.vestingKeeper.IterateVestingAccounts(ctx, func(vestingAcc vestexported.VestingAccount) bool {
		vestingCoins := vestingAcc.GetVestingCoins(ctx.BlockTime())
		for _, coin := range vestingCoins {
			if coin.Denom == k.BondDenom(ctx) {
				totalLocked = totalLocked.Add(coin.Amount)
				break
			} else {
				ctx.Logger().With("module", "x/staking").Info("non-staking token",
					"acc", vestingAcc,
					"coin", coin)
			}
		}
		return false
	})

	// k.authKeeper.IterateAccounts(ctx, func(acc authtypes.AccountI) bool {
	// 	if vestingAcc, ok := acc.(vestexported.VestingAccount); ok {
	// 		vestingCoins := vestingAcc.GetVestingCoins(ctx.BlockTime())
	// 		for _, coin := range vestingCoins {
	// 			if coin.Denom == k.BondDenom(ctx) {
	// 				totalLocked = totalLocked.Add(coin.Amount)
	// 			}
	// 		}
	// 	}
	// 	return false
	// })

	return totalSupply.Sub(totalLocked)
}

// BondedRatio the fraction of the staking tokens which are currently bonded
func (k Keeper) BondedRatio(ctx sdk.Context) sdk.Dec {
	stakeSupply := k.StakingTokenSupply(ctx)
	if stakeSupply.IsPositive() {
		return sdk.NewDecFromInt(k.TotalBondedTokens(ctx)).QuoInt(stakeSupply)
	}

	return sdk.ZeroDec()
}
