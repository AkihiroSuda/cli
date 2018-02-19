package system

import (
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
)

type dialStdioOptions struct {
}

// newDialStdioCommand creates a new cobra.Command for `docker system dial-stdio`
func newDialStdioCommand(dockerCli command.Cli) *cobra.Command {
	var opts dialStdioOptions

	cmd := &cobra.Command{
		Use:    "dial-stdio [OPTIONS]",
		Short:  "Proxy the stdio stream to the daemon connection. Should not be invoked manually.",
		Args:   cli.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDialStdio(dockerCli, opts)
		},
	}

	return cmd
}

func runDialStdio(dockerCli command.Cli, opts dialStdioOptions) error {
	ctx := context.Background()
	conn, err := dockerCli.Client().DialRaw(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to open the raw stream connection")
	}
	connHalfCloser, ok := conn.(halfCloser)
	if !ok {
		return errors.New("the raw stream connection does not implement halfCloser.")
	}
	stdio := &cliHalfCloser{dockerCli}

	var g errgroup.Group
	g.Go(func() error { return copier(stdio, connHalfCloser) })
	g.Go(func() error { return copier(connHalfCloser, stdio) })
	return g.Wait()
}

func copier(to halfCloser, from halfCloser) error {
	if _, err := io.Copy(to, from); err != nil {
		return errors.Wrapf(err, "error while Copy(to=%+v, from=%+v)", to, from)
	}
	if err := from.CloseRead(); err != nil {
		return errors.Wrapf(err, "error while CloseRead(from=%+v)", from)
	}
	if err := to.CloseWrite(); err != nil {
		return errors.Wrapf(err, "error while CloseWrite(to=%+v)", to)
	}
	return nil
}

type halfReadCloser interface {
	io.Reader
	CloseRead() error
}

type halfWriteCloser interface {
	io.Writer
	CloseWrite() error
}

type halfCloser interface {
	halfReadCloser
	halfWriteCloser
}

type cliHalfCloser struct {
	command.Cli
}

func (x *cliHalfCloser) Read(p []byte) (int, error) {
	return x.In().Read(p)
}

func (x *cliHalfCloser) CloseRead() error {
	return x.In().Close()
}

func (x *cliHalfCloser) Write(p []byte) (int, error) {
	return x.Out().Write(p)
}

func (x *cliHalfCloser) CloseWrite() error {
	return x.Out().Close()
}
