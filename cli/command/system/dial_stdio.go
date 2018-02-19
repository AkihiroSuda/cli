package system

import (
	"context"
	"io"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type dialStdioOptions struct {
	// equivalent of `socat -t` and `nc -q`
	timeout time.Duration
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

	flags := cmd.Flags()
	flags.DurationVarP(&opts.timeout, "timeout", "t", 500*time.Millisecond, "After EOF from one of inputs, wait for this duration before exit")
	return cmd
}

func runDialStdio(dockerCli command.Cli, opts dialStdioOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := dockerCli.Client().DialRaw(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to open the raw stream connection")
	}
	connHalfCloser, ok := conn.(halfCloser)
	if !ok {
		return errors.New("the raw stream connection does not implement halfCloser.")
	}
	stdio := &cliHalfCloser{dockerCli}
	stdin2conn := make(chan error)
	conn2stdout := make(chan error)

	go func() {
		err := copier(connHalfCloser, stdio)
		stdin2conn <- err
	}()
	go func() {
		err := copier(stdio, connHalfCloser)
		conn2stdout <- err
	}()
	select {
	case err := <-stdin2conn:
		logrus.Debugf("stdin2conn: %v", err)
	case err := <-conn2stdout:
		logrus.Debugf("conn2stdout: %v", err)
	}
	select {
	case <-time.After(opts.timeout):
		logrus.Debugf("timedout after %v", opts.timeout)
	case err := <-stdin2conn:
		logrus.Debugf("stdin2conn: %v", err)
	case err := <-conn2stdout:
		logrus.Debugf("conn2stdout: %v", err)
	}
	return nil
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
