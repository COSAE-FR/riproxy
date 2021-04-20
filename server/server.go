package server

import (
	"bytes"
	"context"
	"fmt"
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/utils"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Server struct {
	Interface      configuration.InterfaceConfig
	Listener       *net.TCPListener
	Http           *http.Server
	Log            *log.Entry
	WpadFile       string
	ReverseProxies map[string]httputil.ReverseProxy
	Proxy          *ProxyServer
}

func (d Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip, port := utils.GetConnection(r.RemoteAddr)
	logger := d.Log.WithFields(log.Fields{
		"src":         ip.String(),
		"src_port":    port,
		"http_method": r.Method,
		"uri_path":    r.URL.Path,
		"url":         r.URL.String(),
	})
	proxy, ok := d.ReverseProxies[r.Host]
	if ok {
		logger = logger.WithFields(log.Fields{
			"component": "reverse",
			"host":      r.Host,
		})
		logger.Info("reverse proxying")
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
			logger.Errorf("error with reverse proxy: %s", err)
			writer.WriteHeader(http.StatusBadGateway)
		}
		proxy.ServeHTTP(w, r)
		return
	}
	if r.Method == "GET" {
		if WpadPaths[r.URL.Path] {
			logger.WithFields(log.Fields{
				"component": "wpad",
				"status":    200,
			}).Infof("WPAD request %s", r.URL.Path)
			w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
			_, _ = fmt.Fprint(w, d.WpadFile)
		} else {
			logger.WithFields(log.Fields{
				"type":   "wpad",
				"status": 404,
			}).Errorf("Wrong WPAD request %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		logger.Warnf("incorrect method")
		w.WriteHeader(http.StatusNotFound)
	}

}

func (d *Server) Start() error {
	d.Log.Debug("starting HTTP daemon")
	go d.Http.Serve(d.Listener)
	if d.Interface.EnableProxy && d.Proxy != nil {
		d.Proxy.Start()
	}
	return nil
}

func (d Server) Stop() error {
	var err error
	d.Log.Debugf("stopping HTTP daemon")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = d.Http.Shutdown(ctx); err != nil {
		d.Log.Errorf("shutdown error: %v", err)
	} else {
		d.Log.Debug("gracefully stopped")
	}
	if d.Interface.EnableProxy && d.Proxy != nil {
		d.Proxy.Stop()
	}
	return err
}

func New(iface configuration.InterfaceConfig, logger *log.Entry) (*Server, error) {
	la, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", iface.BindIP, iface.BindPort))
	if err != nil {
		logger.Errorf("cannot parse bind address for %s", iface.Name)
		return nil, err
	}
	listener, err := net.ListenTCP("tcp4", la)
	if err != nil {
		logger.Errorf("cannot bind address for %s", iface.Name)
		return nil, err
	}
	buf := new(bytes.Buffer)
	svr := Server{
		Interface: iface,
		Log:       logger,
		Listener:  listener,
		Http:      &http.Server{},
	}
	err = wpadFile.Execute(buf, iface)
	if err != nil {
		logger.Errorf("cannot execute WPAD template; %s", err)
		return nil, err
	}
	svr.WpadFile = buf.String()

	svr.Http.Handler = &svr
	svr.ReverseProxies = make(map[string]httputil.ReverseProxy)
	for name, config := range iface.ReverseProxy {
		destination := config.Destination
		if config.DestinationIp != nil {
			destination = config.DestinationIp.String()
		}
		targetUrl, _ := url.Parse(fmt.Sprintf("http://%s:%d/", destination, config.DestinationPort))
		srcAddr := &net.TCPAddr{
			IP: config.SourceIP,
		}
		logger.Debugf("Setting source ip to %s", config.SourceIP.String())
		transport := &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				LocalAddr: srcAddr,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		proxy := httputil.NewSingleHostReverseProxy(targetUrl)
		proxy.Transport = transport

		svr.ReverseProxies[name] = *proxy
	}
	if iface.EnableProxy {
		svr.Proxy, err = NewProxy(iface, logger)
		if err != nil {
			logger.Errorf("cannot create HTTP Proxy server: %s", err)
			return nil, err
		}
	}
	return &svr, nil
}
