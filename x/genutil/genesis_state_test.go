package genutil

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func TestSanitize(t *testing.T) {
	genesisState := makeGenesisState(t, nil)
	require.Nil(t, mbm.ValidateGenesis(genesisState.Modules))

	addr1 := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	authAcc1 := auth.NewBaseAccountWithAddress(addr1)
	authAcc1.SetCoins(sdk.Coins{
		sdk.NewInt64Coin("bcoin", 150),
		sdk.NewInt64Coin("acoin", 150),
	})
	authAcc1.SetAccountNumber(1)
	genAcc1 := NewGenesisAccount(&authAcc1)

	addr2 := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	authAcc2 := auth.NewBaseAccountWithAddress(addr2)
	authAcc2.SetCoins(sdk.Coins{
		sdk.NewInt64Coin("acoin", 150),
		sdk.NewInt64Coin("bcoin", 150),
	})
	genAcc2 := NewGenesisAccount(&authAcc2)

	genesisState.Accounts = []GenesisAccount{genAcc1, genAcc2}
	require.True(t, genesisState.Accounts[0].AccountNumber > genesisState.Accounts[1].AccountNumber)
	require.Equal(t, genesisState.Accounts[0].Coins[0].Denom, "bcoin")
	require.Equal(t, genesisState.Accounts[0].Coins[1].Denom, "acoin")
	require.Equal(t, genesisState.Accounts[1].Address, addr2)
	genesisState.Sanitize()
	require.False(t, genesisState.Accounts[0].AccountNumber > genesisState.Accounts[1].AccountNumber)
	require.Equal(t, genesisState.Accounts[1].Address, addr1)
	require.Equal(t, genesisState.Accounts[1].Coins[0].Denom, "acoin")
	require.Equal(t, genesisState.Accounts[1].Coins[1].Denom, "bcoin")
}

var (
	pk1   = ed25519.GenPrivKey().PubKey()
	pk2   = ed25519.GenPrivKey().PubKey()
	pk3   = ed25519.GenPrivKey().PubKey()
	addr1 = sdk.ValAddress(pk1.Address())
	addr2 = sdk.ValAddress(pk2.Address())
	addr3 = sdk.ValAddress(pk3.Address())
)

func makeGenesisState(t *testing.T, genTxs []auth.StdTx) GenesisState {
	// start with the default staking genesis state
	appState := NewDefaultGenesisState()
	genAccs := make([]GenesisAccount, len(genTxs))

	cdc := MakeCodec()
	stakingDataBz := appState.Modules[staking.ModuleName]
	var stakingData staking.GenesisState
	cdc.MustUnmarshalJSON(stakingDataBz, &stakingData)

	for i, genTx := range genTxs {
		msgs := genTx.GetMsgs()
		require.Equal(t, 1, len(msgs))
		msg := msgs[0].(staking.MsgCreateValidator)

		acc := auth.NewBaseAccountWithAddress(sdk.AccAddress(msg.ValidatorAddress))
		acc.Coins = sdk.NewCoins(sdk.NewInt64Coin(testBondDenom, 150))
		genAccs[i] = NewGenesisAccount(&acc)
		stakingData.Pool.NotBondedTokens = stakingData.Pool.NotBondedTokens.Add(sdk.NewInt(150)) // increase the supply
	}
	stakingDataBz = cdc.MustMarshalJSON(stakingData)
	appState.Modules[staking.ModuleName] = stakingDataBz

	// create the final app state
	appState.Accounts = genAccs
	return appState
}

// TODO delete
func makeMsg(name string, pk crypto.PubKey) auth.StdTx {
	desc := staking.NewDescription(name, "", "", "")
	comm := staking.CommissionMsg{}
	msg := staking.NewMsgCreateValidator(sdk.ValAddress(pk.Address()), pk, sdk.NewInt64Coin(sdk.DefaultBondDenom,
		50), desc, comm, sdk.OneInt())
	return auth.NewStdTx([]sdk.Msg{msg}, auth.StdFee{}, nil, "")
}

// XXX make depend only on module genesis state
func TestGaiaGenesisValidation(t *testing.T) {
	genTxs := []auth.StdTx{makeMsg("test-0", pk1), makeMsg("test-1", pk2)}
	dupGenTxs := []auth.StdTx{makeMsg("test-0", pk1), makeMsg("test-1", pk1)}
	cdc := MakeCodec()

	// require duplicate accounts fails validation
	genesisState := makeGenesisState(t, dupGenTxs)
	err := mbm.ValidateGenesis(genesisState.Modules)
	require.Error(t, err)

	// require invalid vesting account fails validation (invalid end time)
	genesisState = makeGenesisState(t, genTxs)
	genesisState.Accounts[0].OriginalVesting = genesisState.Accounts[0].Coins
	err = mbm.ValidateGenesis(genesisState.Modules)
	require.Error(t, err)
	genesisState.Accounts[0].StartTime = 1548888000
	genesisState.Accounts[0].EndTime = 1548775410
	err = mbm.ValidateGenesis(genesisState.Modules)
	require.Error(t, err)

	// require bonded + jailed validator fails validation
	genesisState = makeGenesisState(t, genTxs)
	val1 := staking.NewValidator(addr1, pk1, staking.NewDescription("test #2", "", "", ""))
	val1.Jailed = true
	val1.Status = sdk.Bonded

	stakingDataBz := genesisState.Modules[staking.ModuleName]
	var stakingData staking.GenesisState
	cdc.MustUnmarshalJSON(stakingDataBz, &stakingData)
	stakingData.Validators = append(stakingData.Validators, val1)
	stakingDataBz = cdc.MustMarshalJSON(stakingData)
	genesisState.Modules[staking.ModuleName] = stakingDataBz
	err = mbm.ValidateGenesis(genesisState.Modules)
	require.Error(t, err)

	// require duplicate validator fails validation
	val1.Jailed = false
	genesisState = makeGenesisState(t, genTxs)
	val2 := staking.NewValidator(addr1, pk1, staking.NewDescription("test #3", "", "", ""))
	stakingDataBz = genesisState.Modules[staking.ModuleName]
	cdc.MustUnmarshalJSON(stakingDataBz, &stakingData)
	stakingData.Validators = append(stakingData.Validators, val1)
	stakingData.Validators = append(stakingData.Validators, val2)
	stakingDataBz = cdc.MustMarshalJSON(stakingData)
	genesisState.Modules[staking.ModuleName] = stakingDataBz
	err = mbm.ValidateGenesis(genesisState.Modules)
	require.Error(t, err)
}
