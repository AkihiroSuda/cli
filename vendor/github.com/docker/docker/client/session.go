package client // import "github.com/docker/docker/client"

import (
	"net"
	"net/http"

	"golang.org/x/net/context"
)

// DialSession returns a connection that can be used communication with daemon
func (cli *Client) DialSession(ctx context.Context, proto string, meta map[string][]string) (net.Conn, error) {
	req, err := http.NewRequest("POST", "/session", nil)
	if err != nil {
		return nil, err
	}
	req = cli.addHeaders(req, meta)

	return cli.setupHijackConn(ctx, req, proto)
}

// DialRaw returns a raw stream connection, with HTTP/1.1 header, that can be used for proxying the daemon connection.
// Used by `docker dial-stdio` (docker/cli#889).
func (cli *Client) DialRaw(ctx context.Context) (net.Conn, error) {
	return cli.hijackDialer(ctx, cli.proto, cli.addr)
}
