package winrmtest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
)

type Remote struct {
	Host    string
	Port    int
	server  *httptest.Server
	service *wsman
}

func NewRemote() *Remote {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)

	host, port, _ := splitAddr(srv.URL)
	remote := Remote{
		Host:    host,
		Port:    port,
		server:  srv,
		service: &wsman{},
	}

	mux.Handle("/wsman", remote.service)
	return &remote
}

func (r *Remote) Close() {
	r.server.Close()
}

type CommandFunc func(out, err io.Writer) (exitCode int)

func (r *Remote) CommandFunc(cmd string, f CommandFunc) {
	r.service.HandleCommand(cmd, f)
}

func splitAddr(addr string) (host string, port int, err error) {
	u, err := url.Parse(addr)
	if err != nil {
		return
	}

	split := strings.Split(u.Host, ":")
	host = split[0]
	port, err = strconv.Atoi(split[1])
	return
}
