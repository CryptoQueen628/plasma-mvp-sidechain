package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AdityaSripal/plasma-mvp-sidechain/app"
	plasmacfg "github.com/AdityaSripal/plasma-mvp-sidechain/client/plasmad/config"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/cli"
	"github.com/tendermint/tendermint/libs/common"
)

const (
	flagOverwrite  = "overwrite"
	flagClientHome = "home-client"
	flagMoniker    = "moniker"
)

type printInfo struct {
	Moniker    string          `json:"moniker"`
	ChainID    string          `json:"chain_id"`
	NodeID     string          `json:"node_id"`
	AppMessage json.RawMessage `json:"app_message"`
}

// nolint: errcheck
func displayInfo(cdc *codec.Codec, info printInfo) error {
	out, err := codec.MarshalJSONIndent(cdc, info)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s\n", string(out))
	return nil
}

// get cmd to initialize all files for tendermint and application
// nolint
func InitCmd(ctx *server.Context, cdc *codec.Codec, appInit server.AppInit) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long:  `Initialize validators's and node's configuration files.`,
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("test-chain-%v", common.RandStr(6))
			}
			nodeID, _, err := InitializeNodeValidatorFiles(config)
			if err != nil {
				return err
			}

			if viper.GetString(flagMoniker) != "" {
				config.Moniker = viper.GetString(flagMoniker)
			}
			valPubKey := ReadOrCreatePrivValidator(config.PrivValidatorFile())

			var appState json.RawMessage
			genFile := config.GenesisFile()
			if appState, err = initializeEmptyGenesis(cdc, genFile, chainID, valPubKey,
				viper.GetBool(flagOverwrite)); err != nil {
				return err
			}

			if err = ExportGenesisFile(genFile, chainID, nil, appState); err != nil {
				return err
			}

			toPrint := printInfo{
				ChainID:    chainID,
				Moniker:    config.Moniker,
				NodeID:     nodeID,
				AppMessage: appState,
			}

			cfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "config.toml"), config)

			plasmaConfig := plasmacfg.DefaultConfig()
			plasmacfg.WriteConfigFile(filepath.Join(config.RootDir, "config", "plasma.toml"), plasmaConfig)

			fmt.Printf("Add an ethereum address to 'fee_address' to collect fees as a validator\n\n")
			return displayInfo(cdc, toPrint)
		},
	}

	cmd.Flags().String(cli.HomeFlag, app.DefaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(flagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(flagMoniker, "", "set the validator's moniker")
	return cmd
}
