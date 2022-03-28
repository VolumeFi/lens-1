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
		dynListMethodsCmd(a),
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
			gRPCAddr, err := cmd.Flags().GetString(addressFlag)
			if err != nil {
				return err
			}

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
	conn, err := dialGRPC(cmd, a, addr)
	if err != nil {
		return err
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

func dynListMethodsCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-methods [CHAIN_ID] SERVICE",
		Aliases: []string{"lm"},
		// Short:   "List remote gRPC endpoints on the specified chain",
		Args: cobra.RangeArgs(1, 2),
		Example: fmt.Sprintf(
			`$ %s dynamic list-methods cosmoshub cosmos.staking.v1beta1.Query
$ %s dynamic list-methods cosmos.staking.v1beta1.Query --address example.com:9090`,
			appName, appName,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			gRPCAddr, err := cmd.Flags().GetString(addressFlag)
			if err != nil {
				return err
			}

			if (gRPCAddr != "" && len(args) > 1) || (gRPCAddr == "" && len(args) == 1) {
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

			path := args[0]
			if len(args) > 1 {
				path = args[1]
			}

			return dynamicListMethods(cmd, a, gRPCAddr, path)
		},
	}

	return gRPCFlags(cmd, a.Viper)
}

func dynamicListMethods(cmd *cobra.Command, a *appState, gRPCAddr, serviceName string) error {
	conn, err := dialGRPC(cmd, a, gRPCAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	a.Log.Debug("Resolving remote service", zap.String("service_name", serviceName))
	stub := rpb.NewServerReflectionClient(conn)
	c := grpcreflect.NewClient(cmd.Context(), stub)

	d, err := c.ResolveService(serviceName)
	if err != nil {
		if strings.Contains(err.Error(), "Service not found") {
			// If we can list the available services, return a more useful error.
			services, svcErr := c.ListServices()
			if svcErr == nil {
				return GRPCServiceNotFoundError{
					Requested: serviceName,
					Available: services,
				}
			}
		}

		return fmt.Errorf("failed to resolve service: %w", err)
	}
	for _, m := range d.GetMethods() {
		fmt.Fprintln(cmd.OutOrStdout(), m.GetName())
	}

	return nil
}

func dialGRPC(cmd *cobra.Command, a *appState, addr string) (*grpc.ClientConn, error) {
	insec, err := cmd.Flags().GetBool(insecureFlag)
	if err != nil {
		return nil, err
	}
	var dialOpts []grpc.DialOption
	if insec {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	a.Log.Debug("Opening remote gRPC connection", zap.String("addr", addr))
	conn, err := grpc.DialContext(cmd.Context(), addr, dialOpts...)
	if err != nil {
		if strings.Contains(err.Error(), "grpc: no transport security set") {
			// Have to use string matching for unexported grpc.errNoTransportSecurity error value.
			a.Log.Warn("Consider using --insecure flag")
		}
		return nil, fmt.Errorf("failed to dial gRPC address %q: %w", addr, err)
	}

	return conn, nil
}
