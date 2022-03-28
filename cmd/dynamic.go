package cmd

import (
	"fmt"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

func dynamicCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dynamic",
		Aliases: []string{"dyn"},
		Short:   "Dynamic integration with remote chains",
	}

	cmd.AddCommand(
		dynListRoutesCmd(a),
	)

	return cmd
}

func dynListRoutesCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-routes [CHAIN_ID]",
		Aliases: []string{"list", "l"},
		Short:   "List remote gRPC endpoints on the specified chain",
		Args:    cobra.RangeArgs(0, 1),
		Example: fmt.Sprintf(
			`$ %s dynamic list-routes cosmoshub
$ %s dynamic list-routes --address example.com:9090`,
			appName, appName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			gRPCAddr := a.Viper.GetString("address")
			if (gRPCAddr != "" && len(args) > 0) || (gRPCAddr == "" && len(args) == 0) {
				return fmt.Errorf("must provide exactly one of CHAIN_ID or --address flag")
			}

			if gRPCAddr == "" {
				chainName := args[0]
				chain, ok := a.Config.Chains[chainName]
				if !ok {
					return ChainNotFoundError{
						Requested: args[0],
						Config:    a.Config,
					}
				}
				gRPCAddr = chain.GRPCAddr
				if gRPCAddr == "" {
					return fmt.Errorf("no gRPC address set for chain %q", chainName)
				}
			}

			return dynamicListRoutes(cmd, a, gRPCAddr)
		},
	}

	return gRPCFlags(cmd, a.Viper)
}

func dynamicListRoutes(cmd *cobra.Command, a *appState, addr string) error {
	var dialOpts []grpc.DialOption
	if a.Viper.GetBool("insecure") {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	a.Log.Debug("Opening remote gRPC connection", zap.String("addr", addr))
	conn, err := grpc.DialContext(cmd.Context(), addr, dialOpts...)
	if err != nil {
		if strings.Contains(err.Error(), "grpc: no transport security set") {
			// Have to use string matching for unexported grpc.errNoTransportSecurity error value.
			a.Log.Warn("Consider using --insecure flag")
		}
		return fmt.Errorf("failed to dial gRPC address %q: %w", addr, err)
	}
	defer conn.Close()

	a.Log.Debug("Listing remote services")
	stub := rpb.NewServerReflectionClient(conn)
	c := grpcreflect.NewClient(cmd.Context(), stub)
	services, err := c.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list remote services: %w", err)
	}
	for _, s := range services {
		fmt.Fprintln(cmd.OutOrStdout(), s)
	}

	return nil
}
