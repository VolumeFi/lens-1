package cmd_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	channelzsvc "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/reflection"
)

func TestDynamicListServices_ChainID(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	_ = sys.MustRun(t, "chains", "edit", "cosmoshub", "grpc-addr", gRPCAddr)

	res := sys.MustRun(t, "dynamic", "list-services", "cosmoshub", "--insecure")
	require.Equal(t, res.Stdout.String(), "grpc.channelz.v1.Channelz\ngrpc.reflection.v1alpha.ServerReflection\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListServices_AddressFlag(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	res := sys.MustRun(t, "dynamic", "list-services", "--insecure", "--address", gRPCAddr)
	require.Equal(t, res.Stdout.String(), "grpc.channelz.v1.Channelz\ngrpc.reflection.v1alpha.ServerReflection\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListServices_Validation(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	t.Run("provide chain name and address", func(t *testing.T) {
		res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-services", "cosmoshub", "--insecure", "--address", "server.invalid:80")
		require.Error(t, res.Err)
		require.Empty(t, res.Stdout.String())
		require.Contains(t, res.Stderr.String(), "must provide exactly one of")
	})

	t.Run("omit both chain name and address", func(t *testing.T) {
		res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-services")
		require.Error(t, res.Err)
		require.Empty(t, res.Stdout.String())
		require.Contains(t, res.Stderr.String(), "must provide exactly one of")
	})
}

func TestDynamicListMethods_ChainID(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	_ = sys.MustRun(t, "chains", "edit", "cosmoshub", "grpc-addr", gRPCAddr)

	res := sys.MustRun(t, "dynamic", "list-methods", "cosmoshub", "grpc.channelz.v1.Channelz", "--insecure")
	require.Equal(t, res.Stdout.String(), "GetTopChannels\nGetServers\nGetServer\nGetServerSockets\nGetChannel\nGetSubchannel\nGetSocket\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListMethods_AddressFlag(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	res := sys.MustRun(t, "dynamic", "list-methods", "--address", gRPCAddr, "--insecure", "grpc.channelz.v1.Channelz")
	require.Equal(t, res.Stdout.String(), "GetTopChannels\nGetServers\nGetServer\nGetServerSockets\nGetChannel\nGetSubchannel\nGetSocket\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListMethods_Validation(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	t.Run("provide chain name and address", func(t *testing.T) {
		res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-methods", "cosmoshub", "grpc.channelz.v1.Channelz", "--insecure", "--address", "server.invalid:80")
		require.Error(t, res.Err)
		require.Empty(t, res.Stdout.String())
		require.Contains(t, res.Stderr.String(), "must provide exactly one of")
	})

	t.Run("omit both chain name and address", func(t *testing.T) {
		res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-methods", "grpc.channelz.v1.Channelz")
		require.Error(t, res.Err)
		require.Empty(t, res.Stdout.String())
		require.Contains(t, res.Stderr.String(), "must provide exactly one of")
	})
}

func runGRPCReflectionServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	srv := grpc.NewServer()
	reflection.Register(srv)                         // Required for reflection.
	channelzsvc.RegisterChannelzServiceToServer(srv) // Arbitrary other built-in gRPC service to confirm reflection behavior.
	go func() {
		srv.Serve(ln)
	}()
	t.Cleanup(srv.Stop)

	return ln.Addr().String()
}
