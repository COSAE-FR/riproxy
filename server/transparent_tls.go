package server

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/COSAE-FR/riputils/arp"
	"github.com/elazarl/goproxy"
	"github.com/inconshreveable/go-vhost"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/url"
	"sync"
)

type TransparentTlsProxy struct {
	Proxy         *goproxy.ProxyHttpServer
	Log           *log.Entry
	LogMacAddress bool
	listener      net.Listener
	stop          chan struct{}
	wg            sync.WaitGroup
}

func NewTransparentTlsProxy(iface configuration.InterfaceConfig, proxy *goproxy.ProxyHttpServer, logMacAddress bool, logger *log.Entry) (*TransparentTlsProxy, error) {
	proxyLogger := logger.WithFields(log.Fields{
		"component": "https_transparent",
		"port":      iface.Proxy.HttpsTransparentPort,
	})
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", iface.Ip.String(), iface.Proxy.HttpsTransparentPort))
	if err != nil {
		proxyLogger.Error("Cannot listen on interface")
		return nil, err
	}
	transparentProxy := TransparentTlsProxy{
		Proxy:         proxy,
		Log:           proxyLogger,
		LogMacAddress: logMacAddress,
		listener:      ln,
		stop:          make(chan struct{}),
	}

	return &transparentProxy, nil
}

func (d *TransparentTlsProxy) Start() error {
	go d.run()
	return nil
}

func (d *TransparentTlsProxy) Stop() error {
	d.Log.Debug("Stopping HTTPS transparent proxy")
	close(d.stop)
	if err := d.listener.Close(); err != nil {
		d.Log.Errorf("Cannot close listener: %s", err)
	}
	d.wg.Wait()
	return nil
}

func (d *TransparentTlsProxy) run() {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		c, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.stop:
				return
			default:
				d.Log.Errorf("Error accepting new connection: %s", err)
				continue
			}
		}
		go func(c net.Conn) {
			d.wg.Add(1)
			defer d.wg.Done()
			ip, port := utils.GetConnection(c.RemoteAddr().String())
			logger := d.Log.WithFields(log.Fields{
				"src_ip":   ip.String(),
				"src_port": port,
			})
			if d.LogMacAddress {
				mac := arp.Search(ip.String())
				if len(mac.MacAddress) > 0 {
					logger = logger.WithField("src_mac", mac.MacAddress)
				}
			}
			tlsConn, err := vhost.TLS(c)
			if err != nil {
				d.Log.Errorf("Error accepting new connection: %s", err)
				return
			}
			logger = logger.WithField("dest_host", tlsConn.Host())
			if tlsConn.Host() == "" {
				logger.Error("Cannot support non-SNI enabled clients")
				_ = c.Close()
				return
			}
			connectReq := &http.Request{
				Method: "CONNECT",
				URL: &url.URL{
					Opaque: tlsConn.Host(),
					Host:   net.JoinHostPort(tlsConn.Host(), "443"),
				},
				Host:       tlsConn.Host(),
				Header:     make(http.Header),
				RemoteAddr: c.RemoteAddr().String(),
			}
			resp := dumbResponseWriter{tlsConn}
			logger.Debug("Transferring request to proxy")
			d.Proxy.ServeHTTP(resp, connectReq)
		}(c)
	}
}

type dumbResponseWriter struct {
	net.Conn
}

func (dumb dumbResponseWriter) Header() http.Header {
	panic("Header() should not be called on this ResponseWriter")
}

func (dumb dumbResponseWriter) Write(buf []byte) (int, error) {
	if bytes.Equal(buf, []byte("HTTP/1.0 200 OK\r\n\r\n")) {
		return len(buf), nil // throw away the HTTP OK response from the faux CONNECT request
	}
	return dumb.Conn.Write(buf)
}

func (dumb dumbResponseWriter) WriteHeader(code int) {
	panic("WriteHeader() should not be called on this ResponseWriter")
}

func (dumb dumbResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return dumb, bufio.NewReadWriter(bufio.NewReader(dumb), bufio.NewWriter(dumb)), nil
}
