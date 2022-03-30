package byop

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
)

var _ module.AppModuleBasic = Module{}

type Module struct {
	moduleName string

	msgs []proto.Message
}

func NewModule(moduleName string, msgs ...proto.Message) Module {
	return Module{
		moduleName: moduleName,
		msgs:       msgs,
	}
}

func (m Module) Name() string { return m.moduleName }

// RegisterInterfaces is the only method that we care about.
func (m Module) RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		m.msgs...,
	)
}
func (m Module) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) {}

func (m Module) DefaultGenesis(codec.JSONCodec) json.RawMessage {
	panic("not required")
}

func (m Module) ValidateGenesis(codec.JSONCodec, client.TxEncodingConfig, json.RawMessage) error {
	panic("not required")
}

func (m Module) RegisterRESTRoutes(client.Context, *mux.Router) { panic("not required") }

func (m Module) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) { panic("not required") }

func (m Module) GetTxCmd() *cobra.Command { panic("not required") }

func (m Module) GetQueryCmd() *cobra.Command { panic("not required") }
