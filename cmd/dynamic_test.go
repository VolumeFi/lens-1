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

func TestDynamicListRoutes_ChainID(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	_ = sys.MustRun(t, "chains", "edit", "cosmoshub", "grpc-addr", gRPCAddr)

	res := sys.MustRun(t, "dynamic", "list-routes", "cosmoshub", "--insecure")
	require.Equal(t, res.Stdout.String(), "grpc.channelz.v1.Channelz\ngrpc.reflection.v1alpha.ServerReflection\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListRoutes_AddressFlag(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	gRPCAddr := runGRPCReflectionServer(t)

	res := sys.MustRun(t, "dynamic", "list-routes", "--insecure", "--address", gRPCAddr)
	require.Equal(t, res.Stdout.String(), "grpc.channelz.v1.Channelz\ngrpc.reflection.v1alpha.ServerReflection\n")
	require.Empty(t, res.Stderr.String())
}

func TestDynamicListRoutes_RejectBothChainAndAddress(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-routes", "cosmoshub", "--insecure", "--address", "server.invalid:80")
	require.Error(t, res.Err)
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "must provide exactly one of")
}

func TestDynamicListRoutes_RejectMissingChainAndAddress(t *testing.T) {
	t.Parallel()

	sys := NewSystem(t)

	res := sys.Run(zaptest.NewLogger(t), "dynamic", "list-routes")
	require.Error(t, res.Err)
	require.Empty(t, res.Stdout.String())
	require.Contains(t, res.Stderr.String(), "must provide exactly one of")
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
