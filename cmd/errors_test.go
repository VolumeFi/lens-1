package cmd_test

import (
	"testing"

	"github.com/strangelove-ventures/lens/client"
	"github.com/strangelove-ventures/lens/cmd"
	"github.com/stretchr/testify/require"
)

func TestChainNotFoundError(t *testing.T) {
	cfg := &cmd.Config{
		Chains: map[string]*client.ChainClientConfig{
			"foo": nil,
			"bar": nil,
			"baz": nil,
		},
	}

	e := cmd.ChainNotFoundError{
		Requested: "x",
		Config:    cfg,
	}

	// Error message always uses sorted available chain names.
	require.Equal(
		t,
		`no chain "x" found (available chains: bar, baz, foo)`,
		e.Error(),
	)
}
