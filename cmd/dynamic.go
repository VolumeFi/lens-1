package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/types/descriptorpb"
)

func dynamicCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dynamic",
		Aliases: []string{"dyn"},
		Short:   "Dynamic integration with remote chains",
	}

	cmd.AddCommand(
		dynListServicesCmd(a),
		dynListMethodsCmd(a),
		dynShowMessagesCmd(a),
		dynInspectCmd(a),
	)

	return cmd
}

func dynListServicesCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-services [CHAIN_ID]",
		Aliases: []string{"ls"},
		Short:   "List remote gRPC services on the specified chain",
		Args:    cobra.RangeArgs(0, 1),
		Example: fmt.Sprintf(
			`$ %s dynamic list-services cosmoshub
$ %s dynamic list-services --address example.com:9090`,
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

			return dynamicListServices(cmd, a, gRPCAddr)
		},
	}

	return gRPCFlags(cmd, a.Viper)
}

func dynamicListServices(cmd *cobra.Command, a *appState, addr string) error {
	conn, err := dialGRPC(cmd, a, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	a.Log.Debug("Listing remote services")
	stub := rpb.NewServerReflectionClient(conn)
	c := grpcreflect.NewClient(cmd.Context(), stub)
	defer c.Reset()
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
	defer c.Reset()

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

func dynShowMessagesCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show-messages [CHAIN_ID] QUALIFIED_METHOD_NAME",
		Aliases: []string{"sm"},
		// Short:   "List remote gRPC endpoints on the specified chain",
		Args: cobra.RangeArgs(1, 2),
		Example: fmt.Sprintf(
			`$ %s dynamic show-messages cosmoshub cosmos.staking.v1beta1.Query
$ %s dynamic show-messages cosmos.staking.v1beta1.Query --address example.com:9090`,
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

			messageName := args[0]
			if len(args) > 1 {
				messageName = args[1]
			}

			return dynamicShowMessages(cmd, a, gRPCAddr, messageName)
		},
	}

	return gRPCFlags(cmd, a.Viper)
}

func dynamicShowMessages(cmd *cobra.Command, a *appState, gRPCAddr, method string) error {
	conn, err := dialGRPC(cmd, a, gRPCAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	serviceParts := strings.Split(method, ".")
	if len(serviceParts) == 1 {
		return fmt.Errorf("invalid method %q: expected format namespace[.other_namespace...].method", method)
	}
	serviceName := strings.Join(serviceParts[:len(serviceParts)-1], ".")

	a.Log.Debug("Resolving remote service", zap.String("service_name", serviceName))
	stub := rpb.NewServerReflectionClient(conn)
	c := grpcreflect.NewClient(cmd.Context(), stub)
	defer c.Reset()

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

	methodName := serviceParts[len(serviceParts)-1]
	m := d.FindMethodByName(methodName)

	var msgs struct {
		Input  map[string]interface{}
		Output map[string]interface{}
	}

	inType := m.GetInputType()
	if inType != nil {
		msgs.Input = make(map[string]interface{})
		for _, inField := range inType.GetFields() {
			var typeName string
			typ := inField.GetType()
			if typ == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				typeName = inField.GetMessageType().GetFullyQualifiedName()
			} else {
				typeName = descriptorpb.FieldDescriptorProto_Type_name[int32(typ)]
			}
			msgs.Input[inField.GetJSONName()] = typeName
		}
	}
	outType := m.GetOutputType()
	if outType != nil {
		msgs.Output = make(map[string]interface{})
		for _, outField := range outType.GetFields() {
			var typeName string
			typ := outField.GetType()
			if typ == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
				typeName = outField.GetMessageType().GetFullyQualifiedName()
			} else {
				typeName = descriptorpb.FieldDescriptorProto_Type_name[int32(typ)]
			}

			msgs.Output[outField.GetJSONName()] = typeName
		}
	}

	writeJSON(cmd.OutOrStdout(), msgs)

	return nil
}

func dynInspectCmd(a *appState) *cobra.Command {
	const (
		serviceFlag = "service"
		methodFlag  = "method"
	)

	cmd := &cobra.Command{
		Use:     "inspect",
		Aliases: []string{"i"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			gRPCAddr, err := cmd.Flags().GetString(addressFlag)
			if err != nil {
				return err
			}

			serviceName, err := cmd.Flags().GetString(serviceFlag)
			if err != nil {
				return err
			}

			methodName, err := cmd.Flags().GetString(methodFlag)
			if err != nil {
				return err
			}

			a.Log.Debug("Inspecting server", zap.String("addr", gRPCAddr))

			return dynamicInspect(cmd, a, gRPCAddr, serviceName, methodName)
		},
	}

	cmd = gRPCFlags(cmd, a.Viper)

	cmd.Flags().String(serviceFlag, "", "Name of gRPC service to inspect")
	cmd.Flags().String(methodFlag, "", "Name of method within gRPC service to inspect")
	return cmd
}

func dynamicInspect(cmd *cobra.Command, a *appState, gRPCAddr, serviceName, methodName string) error {
	conn, err := dialGRPC(cmd, a, gRPCAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	stub := rpb.NewServerReflectionClient(conn)
	c := grpcreflect.NewClient(cmd.Context(), stub)
	defer c.Reset()

	pp := &protoprint.Printer{
		SortElements:             true,
		ForceFullyQualifiedNames: true,
	}

	if serviceName == "" {
		a.Log.Debug("Listing all services")

		services, err := c.ListServices()
		if err != nil {
			return fmt.Errorf("failed to list remote services: %w", err)
		}

		for _, svc := range services {
			svcDesc, err := c.ResolveService(svc)
			if err != nil {
				a.Log.Info(
					"Error resolving service",
					zap.String("service_name", svcDesc.GetFullyQualifiedName()),
					zap.Error(err),
				)
				continue
			}
			fmt.Fprintln(cmd.OutOrStdout(), svcDesc.GetFullyQualifiedName())
			continue

			proto, err := pp.PrintProtoToString(svcDesc)
			if err != nil {
				a.Log.Info(
					"Error converting to proto string",
					zap.String("service_name", svcDesc.GetFullyQualifiedName()),
					zap.Error(err),
				)
				continue
			}
			fmt.Fprintln(cmd.OutOrStdout(), proto)
		}

		return nil
	}

	a.Log.Debug("Resolving requested service", zap.String("service_name", serviceName))
	svcDesc, err := c.ResolveService(serviceName)
	if err != nil {
		a.Log.Info(
			"Error resolving service",
			zap.String("service_name", svcDesc.GetFullyQualifiedName()),
			zap.Error(err),
		)
		return err
	}

	if methodName == "" {
		proto, err := pp.PrintProtoToString(svcDesc)
		if err != nil {
			a.Log.Info(
				"Error converting to proto string",
				zap.String("service_name", svcDesc.GetFullyQualifiedName()),
				zap.Error(err),
			)
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), proto)

		return nil
	}

	a.Log.Debug("Resolving requested method", zap.String("service_name", serviceName), zap.String("method_name", methodName))
	mDesc := svcDesc.FindMethodByName(methodName)
	if mDesc == nil {
		// TODO: return info about available methods
		return fmt.Errorf("no method with name %q", methodName)
	}

	proto, err := pp.PrintProtoToString(mDesc)
	if err != nil {
		a.Log.Info(
			"Error converting to proto string",
			zap.String("service_name", svcDesc.GetFullyQualifiedName()),
			zap.String("method_name", mDesc.GetFullyQualifiedName()),
			zap.Error(err),
		)
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), proto)

	var s sources
	if inType := mDesc.GetInputType(); inType != nil {
		proto, err := pp.PrintProtoToString(inType)
		if err != nil {
			a.Log.Info(
				"Error converting method input type to string",
				zap.String("service_name", svcDesc.GetFullyQualifiedName()),
				zap.String("method_name", mDesc.GetFullyQualifiedName()),
				zap.Error(err),
			)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "// "+inType.GetFile().GetFullyQualifiedName())
			fmt.Fprintln(cmd.OutOrStdout(), proto)

			s = walkMessageType(inType, s)
		}
	}

	if outType := mDesc.GetOutputType(); outType != nil {
		proto, err := pp.PrintProtoToString(outType)
		if err != nil {
			a.Log.Info(
				"Error converting method output type to string",
				zap.String("service_name", svcDesc.GetFullyQualifiedName()),
				zap.String("method_name", mDesc.GetFullyQualifiedName()),
				zap.Error(err),
			)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "// "+outType.GetFile().GetFullyQualifiedName())
			fmt.Fprintln(cmd.OutOrStdout(), proto)

			s = walkMessageType(outType, s)
		}
	}

	s.Print(a.Log, cmd.OutOrStdout(), pp)

	return nil
}

// sources is a collection of descriptors.
// It is a slice so iteration order when printing is maintained.
type sources []desc.Descriptor

// Contains iterates through the existing sources
// and reports whether it already contains a descriptor matching d's fully qualified name.
func (s sources) Contains(d desc.Descriptor) bool {
	want := d.GetFullyQualifiedName()
	for _, have := range s {
		if have.GetFullyQualifiedName() == want {
			return true
		}
	}

	return false
}

func (s sources) Print(log *zap.Logger, out io.Writer, pp *protoprint.Printer) {
	for _, desc := range s {
		proto, err := pp.PrintProtoToString(desc)
		if err != nil {
			log.Info(
				"Error converting descriptor to string",
				zap.String("fully_qualified_name", desc.GetFullyQualifiedName()),
				zap.Error(err),
			)
			continue
		}

		fmt.Fprintf(out, "// %s (%s)\n", desc.GetFullyQualifiedName(), desc.GetFile().GetFullyQualifiedName())
		fmt.Fprintln(out, proto)
	}
}

func walkMessageType(msgDesc *desc.MessageDescriptor, s sources) sources {
	for _, fDesc := range msgDesc.GetFields() {
		if mDesc := fDesc.GetMessageType(); mDesc != nil {
			if !s.Contains(mDesc) {
				s = append(s, mDesc)
				s = walkMessageType(mDesc, s)
			}

			continue
		}

		if eDesc := fDesc.GetEnumType(); eDesc != nil {
			if !s.Contains(eDesc) {
				s = append(s, eDesc)
				// Enums are just lists of constants, so no need to descend into them.
			}

			continue
		}
	}

	return s
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
