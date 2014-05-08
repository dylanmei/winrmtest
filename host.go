package winrmtest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
)

type Host struct {
	Hostname string
	Port     int
	server   *httptest.Server
	service  *wsman
}

func NewHost() *Host {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)

	hostname, port, _ := splitAddr(srv.URL)
	host := Host{
		Hostname: hostname,
		Port:     port,
		server:   srv,
		service:  &wsman{},
	}

	fmt.Printf("winrmtest listening at %s:%d\n", hostname, port)

	mux.Handle("/wsman", host.service)
	return &host
}

func (h *Host) Close() {
	fmt.Println("winrmtest closing")
	h.server.Close()
}

type CommandFunc func(out, err io.Writer) (exitCode int)

func (h *Host) CommandFunc(cmd string, f CommandFunc) {
	h.service.mapCommandTextToFunc(cmd, f)
}

func splitAddr(hostUrl string) (hostname string, port int, err error) {
	u, err := url.Parse(hostUrl)
	if err != nil {
		return
	}

	split := strings.Split(u.Host, ":")
	hostname = split[0]
	port, err = strconv.Atoi(split[1])
	return
}
