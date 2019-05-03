package client

import (
	"github.com/cosmos/cosmos-sdk/client"
	nftcmd "github.com/cosmos/cosmos-sdk/x/nft/client/cli"
	"github.com/spf13/cobra"

	amino "github.com/tendermint/go-amino"
)

// ModuleClient exports all client functionality from this module
type ModuleClient struct {
	storeKey string
	cdc      *amino.Codec
}

// NewModuleClient creates a new module client
func NewModuleClient(storeKey string, cdc *amino.Codec) ModuleClient {
	return ModuleClient{storeKey, cdc}
}

// GetQueryCmd returns the cli query commands for this module
func (mc ModuleClient) GetQueryCmd() *cobra.Command {
	// Group nameservice queries under a subcommand
	nftQueryCmd := &cobra.Command{
		Use:   "nft",
		Short: "Querying commands for the NFT module",
	}

	nftQueryCmd.AddCommand(client.GetCommands(
		nftcmd.GetCmdQueryCollectionSupply(mc.storeKey, mc.cdc),
		nftcmd.GetCmdQueryBalance(mc.storeKey, mc.cdc),
		nftcmd.GetCmdQueryNFTs(mc.storeKey, mc.cdc),
		nftcmd.GetCmdQueryNFT(mc.storeKey, mc.cdc),
	)...)

	return nftQueryCmd
}

// GetTxCmd returns the transaction commands for this module
func (mc ModuleClient) GetTxCmd() *cobra.Command {
	nftTxCmd := &cobra.Command{
		Use:   "nft",
		Short: "NFT transactions subcommands",
	}

	nftTxCmd.AddCommand(client.PostCommands(
		nftcmd.GetCmdTransferNFT(mc.cdc),
		nftcmd.GetCmdEditNFTMetadata(mc.cdc),
	)...)

	return nftTxCmd
}
