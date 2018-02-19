package system

import (
	"context"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dialer := dockerCli.Client().Dialer()
	conn, err := dialer(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to open the raw stream connection")
	}
	connHalfCloser, ok := conn.(halfCloser)
	if !ok {
		return errors.New("the raw stream connection does not implement halfCloser")
	}
	stdio := &stdioHalfCloser{r: os.Stdin, w: os.Stdout}
	var eg errgroup.Group
	eg.Go(func() error { return copier(connHalfCloser, stdio) })
	eg.Go(func() error { return copier(stdio, connHalfCloser) })
	return eg.Wait()
}

func copier(to halfCloser, from halfCloser) error {
	if _, err := io.Copy(to, from); err != nil {
		return errors.Wrapf(err, "error while Copy (to=%+v, from=%+v)", to, from)
	}
	if err := from.CloseRead(); err != nil {
		return errors.Wrapf(err, "error while CloseRead (from=%+v)", from)
	}
	if err := to.CloseWrite(); err != nil {
		return errors.Wrapf(err, "error while CloseWrite (to=%+v)", to)
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

type stdioHalfCloser struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (x *stdioHalfCloser) Read(p []byte) (int, error) {
	return x.r.Read(p)
}

func (x *stdioHalfCloser) CloseRead() error {
	return x.r.Close()
}

func (x *stdioHalfCloser) Write(p []byte) (int, error) {
	return x.w.Write(p)
}

func (x *stdioHalfCloser) CloseWrite() error {
	return x.w.Close()
}
