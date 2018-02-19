package connhelper

import (
	"context"
	"net"
	"net/url"

	"github.com/pkg/errors"
)

func newSSHConnectionHelper(daemonURL string) (*ConnectionHelper, error) {
	sp, err := parseSSHURL(daemonURL)
	if err != nil {
		return nil, err
	}
	sshBinary := "ssh"
	sshArgs := sp.Args()
	// requires Docker 18.05 or later on the remote host.
	dialStdio := []string{"docker", "system", "dial-stdio"}
	fullArgs := append(sshArgs, append([]string{"--"}, dialStdio...)...)
	h := &ConnectionHelper{
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return newCommandConn(ctx, sshBinary, fullArgs...)
		},
		Host: "http://docker",
	}
	return h, nil
}

func parseSSHURL(daemonURL string) (*sshSpec, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ssh" {
		return nil, errors.Errorf("expected scheme: ssh, got %s", u.Scheme)
	}

	var sp sshSpec

	if u.User != nil {
		sp.user = u.User.Username()
		if _, ok := u.User.Password(); ok {
			return nil, errors.New("ssh does not accept plain-text password")
		}
	}
	sp.host = u.Hostname()
	sp.port = u.Port()
	if u.Path != "" {
		return nil, errors.Errorf("extra path: %s", u.Path)
	}
	if u.RawQuery != "" {
		return nil, errors.Errorf("extra query: %s", u.RawQuery)
	}
	if u.Fragment != "" {
		return nil, errors.Errorf("extra fragment: %s", u.Fragment)
	}
	return &sp, err
}

type sshSpec struct {
	user string
	host string
	port string
}

func (sp *sshSpec) Args() []string {
	var args []string
	if sp.user != "" {
		args = append(args, "-l", sp.user)
	}
	if sp.port != "" {
		args = append(args, "-p", sp.port)
	}
	args = append(args, sp.host)
	return args
}
