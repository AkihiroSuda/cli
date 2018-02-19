// Package connhelper provides helpers for connecting to a remote daemon host with custom logic.
package connhelper

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionHelper allows to connect to a remote host with custom stream provider binary.
type ConnectionHelper struct {
	Dialer    func(ctx context.Context, network, addr string) (net.Conn, error)
	DummyHost string // dummy URL used for HTTP requests. e.g. "http://docker"
}

// GetConnectionHelper returns Docker-specific connection helper for the given URL.
// GetConnectionHelper returns nil without error when no helper is registered for the scheme.
// URL is like "ssh://me@server01".
func GetConnectionHelper(daemonURL string) (*ConnectionHelper, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, err
	}
	switch scheme := u.Scheme; scheme {
	case "ssh":
		return newSSHConnectionHelper(daemonURL)
	}
	// Future version may support plugins via ~/.docker/config.json. e.g. "dind"
	// See docker/cli#889 for the previous discussion.
	return nil, err
}

func newCommandConn(ctx context.Context, cmd string, args ...string) (net.Conn, error) {
	var (
		c   commandConn
		err error
	)
	c.cmd = exec.CommandContext(ctx, cmd, args...)
	// we assume that args never contains sensitive information
	logrus.Debugf("connhelper (%s): starting with %v", cmd, args)
	c.cmd.Env = os.Environ()
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c.cmd.Stderr = &logrusDebugWriter{
		prefix: fmt.Sprintf("connhelper (%s):", cmd),
	}
	c.localAddr = dummyAddr{network: "dummy", s: "dummy-0"}
	c.remoteAddr = dummyAddr{network: "dummy", s: "dummy-1"}
	return &c, c.cmd.Start()
}

// commandConn implements net.Conn
type commandConn struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *commandConn) CloseRead() error {
	return c.stdout.Close()
}

func (c *commandConn) Read(p []byte) (int, error) {
	return c.stdout.Read(p)
}

func (c *commandConn) CloseWrite() error {
	return c.stdin.Close()
}

func (c *commandConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *commandConn) Close() error {
	if err := c.stdin.Close(); err != nil {
		logrus.Warnf("error while closing stdin: %v", err)
	}
	if err := c.stdout.Close(); err != nil {
		logrus.Warnf("error while closing stdout: %v", err)
	}
	if err := c.cmd.Process.Kill(); err != nil {
		return err
	}
	_, err := c.cmd.Process.Wait()
	return err
}

func (c *commandConn) LocalAddr() net.Addr {
	return c.localAddr
}
func (c *commandConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}
func (c *commandConn) SetDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetDeadline(%v)", t)
	return nil
}
func (c *commandConn) SetReadDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetReadDeadline(%v)", t)
	return nil
}
func (c *commandConn) SetWriteDeadline(t time.Time) error {
	logrus.Debugf("unimplemented call: SetWriteDeadline(%v)", t)
	return nil
}

type dummyAddr struct {
	network string
	s       string
}

func (d dummyAddr) Network() string {
	return d.network
}

func (d dummyAddr) String() string {
	return d.s
}

type logrusDebugWriter struct {
	prefix string
}

func (w *logrusDebugWriter) Write(p []byte) (int, error) {
	logrus.Debugf("%s%s", w.prefix, string(p))
	return len(p), nil
}
