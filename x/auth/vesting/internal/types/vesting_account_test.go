package types

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmtime "github.com/tendermint/tendermint/types/time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
)

var (
	stakeDenom = "stake"
	feeDenom   = "fee"
)

func TestGetVestedCoinsContVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	cva := NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())

	// require no coins vested in the very beginning of the vesting schedule
	vestedCoins := cva.GetVestedCoins(now)
	require.Nil(t, vestedCoins)

	// require all coins vested at the end of the vesting schedule
	vestedCoins = cva.GetVestedCoins(endTime)
	require.Equal(t, origCoins, vestedCoins)

	// require 50% of coins vested
	vestedCoins = cva.GetVestedCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestedCoins)

	// require 100% of coins vested
	vestedCoins = cva.GetVestedCoins(now.Add(48 * time.Hour))
	require.Equal(t, origCoins, vestedCoins)
}

func TestGetVestingCoinsContVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	cva := NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())

	// require all coins vesting in the beginning of the vesting schedule
	vestingCoins := cva.GetVestingCoins(now)
	require.Equal(t, origCoins, vestingCoins)

	// require no coins vesting at the end of the vesting schedule
	vestingCoins = cva.GetVestingCoins(endTime)
	require.Nil(t, vestingCoins)

	// require 50% of coins vesting
	vestingCoins = cva.GetVestingCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestingCoins)
}

func TestSpendableCoinsContVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	cva := NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())

	// require that there exist no spendable coins in the beginning of the
	// vesting schedule
	spendableCoins := cva.SpendableCoins(now)
	require.Nil(t, spendableCoins)

	// require that all original coins are spendable at the end of the vesting
	// schedule
	spendableCoins = cva.SpendableCoins(endTime)
	require.Equal(t, origCoins, spendableCoins)

	// require that all vested coins (50%) are spendable
	spendableCoins = cva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, spendableCoins)

	// receive some coins
	recvAmt := sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}
	cva.SetCoins(cva.GetCoins().Add(recvAmt))

	// require that all vested coins (50%) are spendable plus any received
	spendableCoins = cva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 100)}, spendableCoins)

	// spend all spendable coins
	cva.SetCoins(cva.GetCoins().Sub(spendableCoins))

	// require that no more coins are spendable
	spendableCoins = cva.SpendableCoins(now.Add(12 * time.Hour))
	require.Nil(t, spendableCoins)
}

func TestTrackDelegationContVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to delegate all vesting coins
	cva := NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	cva.TrackDelegation(now, origCoins)
	require.Equal(t, origCoins, cva.DelegatedVesting)
	require.Nil(t, cva.DelegatedFree)
	require.Nil(t, cva.GetCoins())

	// require the ability to delegate all vested coins
	bacc.SetCoins(origCoins)
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	cva.TrackDelegation(endTime, origCoins)
	require.Nil(t, cva.DelegatedVesting)
	require.Equal(t, origCoins, cva.DelegatedFree)
	require.Nil(t, cva.GetCoins())

	// require the ability to delegate all vesting coins (50%) and all vested coins (50%)
	bacc.SetCoins(origCoins)
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	cva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, cva.DelegatedVesting)
	require.Nil(t, cva.DelegatedFree)

	cva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, cva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, cva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000)}, cva.GetCoins())

	// require no modifications when delegation amount is zero or not enough funds
	bacc.SetCoins(origCoins)
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	require.Panics(t, func() {
		cva.TrackDelegation(endTime, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 1000000)})
	})
	require.Nil(t, cva.DelegatedVesting)
	require.Nil(t, cva.DelegatedFree)
	require.Equal(t, origCoins, cva.GetCoins())
}

func TestTrackUndelegationContVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to undelegate all vesting coins
	cva := NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	cva.TrackDelegation(now, origCoins)
	cva.TrackUndelegation(origCoins)
	require.Nil(t, cva.DelegatedFree)
	require.Nil(t, cva.DelegatedVesting)
	require.Equal(t, origCoins, cva.GetCoins())

	// require the ability to undelegate all vested coins
	bacc.SetCoins(origCoins)
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())

	cva.TrackDelegation(endTime, origCoins)
	cva.TrackUndelegation(origCoins)
	require.Nil(t, cva.DelegatedFree)
	require.Nil(t, cva.DelegatedVesting)
	require.Equal(t, origCoins, cva.GetCoins())

	// require no modifications when the undelegation amount is zero
	bacc.SetCoins(origCoins)
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())

	require.Panics(t, func() {
		cva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 0)})
	})
	require.Nil(t, cva.DelegatedFree)
	require.Nil(t, cva.DelegatedVesting)
	require.Equal(t, origCoins, cva.GetCoins())

	// vest 50% and delegate to two validators
	cva = NewContinuousVestingAccount(&bacc, now.Unix(), endTime.Unix())
	cva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	cva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})

	// undelegate from one validator that got slashed 50%
	cva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)}, cva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, cva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 25)}, cva.GetCoins())

	// undelegate from the other validator that did not get slashed
	cva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Nil(t, cva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)}, cva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 75)}, cva.GetCoins())
}

func TestGetVestedCoinsDelVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require no coins are vested until schedule maturation
	dva := NewDelayedVestingAccount(&bacc, endTime.Unix())
	vestedCoins := dva.GetVestedCoins(now)
	require.Nil(t, vestedCoins)

	// require all coins be vested at schedule maturation
	vestedCoins = dva.GetVestedCoins(endTime)
	require.Equal(t, origCoins, vestedCoins)
}

func TestGetVestingCoinsDelVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require all coins vesting at the beginning of the schedule
	dva := NewDelayedVestingAccount(&bacc, endTime.Unix())
	vestingCoins := dva.GetVestingCoins(now)
	require.Equal(t, origCoins, vestingCoins)

	// require no coins vesting at schedule maturation
	vestingCoins = dva.GetVestingCoins(endTime)
	require.Nil(t, vestingCoins)
}

func TestSpendableCoinsDelVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require that no coins are spendable in the beginning of the vesting
	// schedule
	dva := NewDelayedVestingAccount(&bacc, endTime.Unix())
	spendableCoins := dva.SpendableCoins(now)
	require.Nil(t, spendableCoins)

	// require that all coins are spendable after the maturation of the vesting
	// schedule
	spendableCoins = dva.SpendableCoins(endTime)
	require.Equal(t, origCoins, spendableCoins)

	// require that all coins are still vesting after some time
	spendableCoins = dva.SpendableCoins(now.Add(12 * time.Hour))
	require.Nil(t, spendableCoins)

	// receive some coins
	recvAmt := sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}
	dva.SetCoins(dva.GetCoins().Add(recvAmt))

	// require that only received coins are spendable since the account is still
	// vesting
	spendableCoins = dva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, recvAmt, spendableCoins)

	// spend all spendable coins
	dva.SetCoins(dva.GetCoins().Sub(spendableCoins))

	// require that no more coins are spendable
	spendableCoins = dva.SpendableCoins(now.Add(12 * time.Hour))
	require.Nil(t, spendableCoins)
}

func TestTrackDelegationDelVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to delegate all vesting coins
	bacc.SetCoins(origCoins)
	dva := NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(now, origCoins)
	require.Equal(t, origCoins, dva.DelegatedVesting)
	require.Nil(t, dva.DelegatedFree)
	require.Nil(t, dva.GetCoins())

	// require the ability to delegate all vested coins
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(endTime, origCoins)
	require.Nil(t, dva.DelegatedVesting)
	require.Equal(t, origCoins, dva.DelegatedFree)
	require.Nil(t, dva.GetCoins())

	// require the ability to delegate all coins half way through the vesting
	// schedule
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(now.Add(12*time.Hour), origCoins)
	require.Equal(t, origCoins, dva.DelegatedVesting)
	require.Nil(t, dva.DelegatedFree)
	require.Nil(t, dva.GetCoins())

	// require no modifications when delegation amount is zero or not enough funds
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())

	require.Panics(t, func() {
		dva.TrackDelegation(endTime, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 1000000)})
	})
	require.Nil(t, dva.DelegatedVesting)
	require.Nil(t, dva.DelegatedFree)
	require.Equal(t, origCoins, dva.GetCoins())
}

func TestTrackUndelegationDelVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to undelegate all vesting coins
	bacc.SetCoins(origCoins)
	dva := NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(now, origCoins)
	dva.TrackUndelegation(origCoins)
	require.Nil(t, dva.DelegatedFree)
	require.Nil(t, dva.DelegatedVesting)
	require.Equal(t, origCoins, dva.GetCoins())

	// require the ability to undelegate all vested coins
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(endTime, origCoins)
	dva.TrackUndelegation(origCoins)
	require.Nil(t, dva.DelegatedFree)
	require.Nil(t, dva.DelegatedVesting)
	require.Equal(t, origCoins, dva.GetCoins())

	// require no modifications when the undelegation amount is zero
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())

	require.Panics(t, func() {
		dva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 0)})
	})
	require.Nil(t, dva.DelegatedFree)
	require.Nil(t, dva.DelegatedVesting)
	require.Equal(t, origCoins, dva.GetCoins())

	// vest 50% and delegate to two validators
	bacc.SetCoins(origCoins)
	dva = NewDelayedVestingAccount(&bacc, endTime.Unix())
	dva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	dva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})

	// undelegate from one validator that got slashed 50%
	dva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)})

	require.Nil(t, dva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 75)}, dva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 25)}, dva.GetCoins())

	// undelegate from the other validator that did not get slashed
	dva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Nil(t, dva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)}, dva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 75)}, dva.GetCoins())
}

func TestGetVestedCoinsPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)
	periods := VestingPeriods{
		VestingPeriod{PeriodLength: int64(12 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
	}

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	pva := NewPeriodicVestingAccount(&bacc, now.Unix(), periods)

	// require no coins vested at the beginning of the vesting schedule
	vestedCoins := pva.GetVestedCoins(now)
	require.Nil(t, vestedCoins)

	// require all coins vested at the end of the vesting schedule
	vestedCoins = pva.GetVestedCoins(endTime)
	require.Equal(t, origCoins, vestedCoins)

	// require no coins vested during first vesting period
	vestedCoins = pva.GetVestedCoins(now.Add(6 * time.Hour))
	require.Nil(t, vestedCoins)

	// require 50% of coins vested after period 1
	vestedCoins = pva.GetVestedCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestedCoins)

	// require period 2 coins don't vest until period is over
	vestedCoins = pva.GetVestedCoins(now.Add(15 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestedCoins)

	// require 75% of coins vested after period 2
	vestedCoins = pva.GetVestedCoins(now.Add(18 * time.Hour))
	require.Equal(t,
		sdk.Coins{
			sdk.NewInt64Coin(feeDenom, 750), sdk.NewInt64Coin(stakeDenom, 75)}, vestedCoins)

	// require 100% of coins vested
	vestedCoins = pva.GetVestedCoins(now.Add(48 * time.Hour))
	require.Equal(t, origCoins, vestedCoins)
}

func TestGetVestingCoinsPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)
	periods := VestingPeriods{
		VestingPeriod{PeriodLength: int64(12 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
	}

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{
		sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	pva := NewPeriodicVestingAccount(&bacc, now.Unix(), periods)

	// require all coins vesting at the beginning of the vesting schedule
	vestingCoins := pva.GetVestingCoins(now)
	require.Equal(t, origCoins, vestingCoins)

	// require no coins vesting at the end of the vesting schedule
	vestingCoins = pva.GetVestingCoins(endTime)
	require.Nil(t, vestingCoins)

	// require 50% of coins vesting
	vestingCoins = pva.GetVestingCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestingCoins)

	// require 50% of coins vesting after period 1, but before period 2 completes.
	vestingCoins = pva.GetVestingCoins(now.Add(15 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, vestingCoins)

	// require 25% of coins vesting after period 2
	vestingCoins = pva.GetVestingCoins(now.Add(18 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}, vestingCoins)

	// require 0% of coins vesting after vesting complete
	vestingCoins = pva.GetVestingCoins(now.Add(48 * time.Hour))
	require.Nil(t, vestingCoins)
}

func TestSpendableCoinsPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)
	periods := VestingPeriods{
		VestingPeriod{PeriodLength: int64(12 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
	}

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{
		sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)
	pva := NewPeriodicVestingAccount(&bacc, now.Unix(), periods)

	// require that there exist no spendable coins at the beginning of the
	// vesting schedule
	spendableCoins := pva.SpendableCoins(now)
	require.Nil(t, spendableCoins)

	// require that all original coins are spendable at the end of the vesting
	// schedule
	spendableCoins = pva.SpendableCoins(endTime)
	require.Equal(t, origCoins, spendableCoins)

	// require that all vested coins (50%) are spendable
	spendableCoins = pva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}, spendableCoins)

	// receive some coins
	recvAmt := sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}
	pva.SetCoins(pva.GetCoins().Add(recvAmt))

	// require that all vested coins (50%) are spendable plus any received
	spendableCoins = pva.SpendableCoins(now.Add(12 * time.Hour))
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 100)}, spendableCoins)

	// spend all spendable coins
	pva.SetCoins(pva.GetCoins().Sub(spendableCoins))

	// require that no more coins are spendable
	spendableCoins = pva.SpendableCoins(now.Add(12 * time.Hour))
	require.Nil(t, spendableCoins)
}

func TestTrackDelegationPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)
	periods := VestingPeriods{
		VestingPeriod{PeriodLength: int64(12 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
	}

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to delegate all vesting coins
	pva := NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	pva.TrackDelegation(now, origCoins)
	require.Equal(t, origCoins, pva.DelegatedVesting)
	require.Nil(t, pva.DelegatedFree)
	require.Nil(t, pva.GetCoins())

	// require the ability to delegate all vested coins
	bacc.SetCoins(origCoins)
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	pva.TrackDelegation(endTime, origCoins)
	require.Nil(t, pva.DelegatedVesting)
	require.Equal(t, origCoins, pva.DelegatedFree)
	require.Nil(t, pva.GetCoins())

	// require the ability to delegate all vesting coins (50%) and all vested coins (50%)
	bacc.SetCoins(origCoins)
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	pva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, pva.DelegatedVesting)
	require.Nil(t, pva.DelegatedFree)

	pva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, pva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, pva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000)}, pva.GetCoins())

	// require no modifications when delegation amount is zero or not enough funds
	bacc.SetCoins(origCoins)
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	require.Panics(t, func() {
		pva.TrackDelegation(endTime, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 1000000)})
	})
	require.Nil(t, pva.DelegatedVesting)
	require.Nil(t, pva.DelegatedFree)
	require.Equal(t, origCoins, pva.GetCoins())
}

func TestTrackUndelegationPeriodicVestingAcc(t *testing.T) {
	now := tmtime.Now()
	endTime := now.Add(24 * time.Hour)
	periods := VestingPeriods{
		VestingPeriod{PeriodLength: int64(12 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 500), sdk.NewInt64Coin(stakeDenom, 50)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
		VestingPeriod{PeriodLength: int64(6 * 60 * 60), VestingAmount: sdk.Coins{sdk.NewInt64Coin(feeDenom, 250), sdk.NewInt64Coin(stakeDenom, 25)}},
	}

	_, _, addr := KeyTestPubAddr()
	origCoins := sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 100)}
	bacc := auth.NewBaseAccountWithAddress(addr)
	bacc.SetCoins(origCoins)

	// require the ability to undelegate all vesting coins
	pva := NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	pva.TrackDelegation(now, origCoins)
	pva.TrackUndelegation(origCoins)
	require.Nil(t, pva.DelegatedFree)
	require.Nil(t, pva.DelegatedVesting)
	require.Equal(t, origCoins, pva.GetCoins())

	// require the ability to undelegate all vested coins
	bacc.SetCoins(origCoins)
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)

	pva.TrackDelegation(endTime, origCoins)
	pva.TrackUndelegation(origCoins)
	require.Nil(t, pva.DelegatedFree)
	require.Nil(t, pva.DelegatedVesting)
	require.Equal(t, origCoins, pva.GetCoins())

	// require no modifications when the undelegation amount is zero
	bacc.SetCoins(origCoins)
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)

	require.Panics(t, func() {
		pva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 0)})
	})
	require.Nil(t, pva.DelegatedFree)
	require.Nil(t, pva.DelegatedVesting)
	require.Equal(t, origCoins, pva.GetCoins())

	// vest 50% and delegate to two validators
	pva = NewPeriodicVestingAccount(&bacc, now.Unix(), periods)
	pva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	pva.TrackDelegation(now.Add(12*time.Hour), sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})

	// undelegate from one validator that got slashed 50%
	pva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)})
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)}, pva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)}, pva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 25)}, pva.GetCoins())

	// undelegate from the other validator that did not get slashed
	pva.TrackUndelegation(sdk.Coins{sdk.NewInt64Coin(stakeDenom, 50)})
	require.Nil(t, pva.DelegatedFree)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(stakeDenom, 25)}, pva.DelegatedVesting)
	require.Equal(t, sdk.Coins{sdk.NewInt64Coin(feeDenom, 1000), sdk.NewInt64Coin(stakeDenom, 75)}, pva.GetCoins())
}

func TestGenesisAccountValidate(t *testing.T) {
	pubkey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubkey.Address())
	baseAcc := auth.NewBaseAccount(addr, nil, pubkey, 0, 0)
	baseAccWithCoins := auth.NewBaseAccount(addr, nil, pubkey, 0, 0)
	baseAccWithCoins.SetCoins(sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)})
	tests := []struct {
		name   string
		acc    authexported.GenesisAccount
		expErr error
	}{
		{
			"valid base account",
			baseAcc,
			nil,
		},
		{
			"invalid base valid account",
			auth.NewBaseAccount(addr, sdk.NewCoins(), secp256k1.GenPrivKey().PubKey(), 0, 0),
			errors.New("pubkey and address pair is invalid"),
		},
		{
			"valid base vesting account",
			NewBaseVestingAccount(baseAcc, sdk.NewCoins(), 100),
			nil,
		},
		{
			"invalid vesting amount; empty Coins",
			NewBaseVestingAccount(
				auth.NewBaseAccount(addr, sdk.NewCoins(), pubkey, 0, 0),
				sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}, 100,
			),
			errors.New("vesting amount cannot be greater than total amount"),
		},
		{
			"invalid vesting amount; OriginalVesting > Coins",
			NewBaseVestingAccount(
				auth.NewBaseAccount(addr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10)), pubkey, 0, 0),
				sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}, 100,
			),
			errors.New("vesting amount cannot be greater than total amount"),
		},
		{
			"invalid vesting amount with multi coins",
			NewBaseVestingAccount(
				auth.NewBaseAccount(addr, sdk.NewCoins(sdk.NewInt64Coin("uatom", 50), sdk.NewInt64Coin("eth", 50)), pubkey, 0, 0),
				sdk.NewCoins(sdk.NewInt64Coin("uatom", 100), sdk.NewInt64Coin("eth", 20)), 100,
			),
			errors.New("vesting amount cannot be greater than total amount"),
		},
		{
			"valid continuous vesting account",
			NewContinuousVestingAccount(baseAcc, 100, 200),
			nil,
		},
		{
			"invalid vesting times",
			NewContinuousVestingAccount(baseAcc, 1654668078, 1554668078),
			errors.New("vesting start-time cannot be before end-time"),
		},
		{
			"valid periodic vesting account",
			NewPeriodicVestingAccount(baseAccWithCoins, 100, VestingPeriods{VestingPeriod{PeriodLength: int64(50), VestingAmount: sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}}}),
			nil,
		},
		{
			"invalid vesting period lengths",
			NewPeriodicVestingAccountRaw(
				NewBaseVestingAccount(
					auth.NewBaseAccount(addr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)), pubkey, 0, 0),
					sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}, 200,
				),
				100, VestingPeriods{VestingPeriod{PeriodLength: int64(50), VestingAmount: sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}}}),
			errors.New("vesting end time does not match length of all vesting periods"),
		},
		{
			"invalid vesting period amounts",
			NewPeriodicVestingAccountRaw(
				NewBaseVestingAccount(
					auth.NewBaseAccount(addr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)), pubkey, 0, 0),
					sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 50)}, 200,
				),
				100, VestingPeriods{VestingPeriod{PeriodLength: int64(100), VestingAmount: sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 25)}}}),
			errors.New("original vesting coins does not match the sum of all coins in vesting periods"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.acc.Validate()
			require.Equal(t, tt.expErr, err)
		})
	}
}
