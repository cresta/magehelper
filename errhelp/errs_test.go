package errhelp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrInt(t *testing.T) {
	require.NoError(t, SecondErr(fmt.Println("Hello world")))
}

func TestMustString(t *testing.T) {
	require.Equal(t, "hello", MustString("hello", nil))
	require.Panics(t, func() { MustString("", errors.New("bad")) })
}
