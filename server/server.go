package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/COSAE-FR/riputils/arp"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type reverseProxy struct {
	Proxy   httputil.ReverseProxy
	Methods map[string]bool
}

type Server struct {
	Interface      configuration.InterfaceConfig
	Listener       *net.TCPListener
	Http           *http.Server
	Log            *log.Entry
	WpadFile       string
	ReverseProxies map[string]reverseProxy
	Proxy          *ProxyServer
	TransparentTls *TransparentTlsProxy
	LogMacAddress  bool
}

func (d Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip, port := utils.GetConnection(r.RemoteAddr)
	logger := d.Log.WithFields(log.Fields{
		"action":      "pass",
		"src":         ip.String(),
		"src_port":    port,
		"http_method": r.Method,
		"uri_path":    r.URL.Path,
		"url":         r.URL.String(),
	})
	if d.LogMacAddress {
		mac := arp.Search(ip.String())
		if len(mac.MacAddress) > 0 {
			logger = logger.WithField("src_mac", mac.MacAddress)
		}
	}
	proxy, ok := d.ReverseProxies[r.Host]
	if ok {
		logger = logger.WithFields(log.Fields{
			"component": "reverse",
			"host":      r.Host,
		})
		if !proxy.Methods[r.Method] {
			logger.WithField("action", "block").Error("Method blocked by policy")
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprintf(w, "Method %s blocked by policy", r.Method)
			return
		}
		logger.Info("reverse proxying")
		proxy.Proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
			logger.WithField("action", "error").Errorf("error with reverse proxy: %s", err)
			writer.WriteHeader(http.StatusBadGateway)
		}
		proxy.Proxy.ServeHTTP(w, r)
		return
	}
	if d.Interface.EnableWpad {
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
					"action": "error",
				}).Errorf("Wrong WPAD request %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			logger.WithFields(log.Fields{
				"status": 401,
				"type":   "wpad",
				"action": "error",
			}).Warnf("incorrect method")
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		logger.WithFields(log.Fields{
			"status": 404,
			"action": "error",
		}).Errorf("No service for this request %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}

}

func (d *Server) Start() error {
	if d.Interface.ShouldStartHttp() && d.Http != nil {
		if d.Listener == nil {
			d.Log.Error("Mandatory listener not ready")
			return errors.New("missing listener")
		}
		go func() {
			d.Log.Debug("starting HTTP daemon")
			err := d.Http.Serve(d.Listener)
			if err != http.ErrServerClosed {
				d.Log.Debugf("HTTP server stopped with error: %s", err)
			}
		}()
	}
	if d.Interface.EnableProxy && d.Proxy != nil {
		_ = d.Proxy.Start()
		if d.Interface.Proxy.HttpsTransparentPort > 0 && d.TransparentTls != nil {
			_ = d.TransparentTls.Start()
		}
	}
	return nil
}

func (d Server) Stop() error {
	var err error
	if d.Interface.ShouldStartHttp() && d.Http != nil {
		d.Log.Debugf("stopping HTTP daemon")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = d.Http.Shutdown(ctx); err != nil {
			d.Log.Errorf("HTTP server shutdown error: %v", err)
		} else {
			d.Log.Debug("HTTP daemon gracefully stopped")
		}
	}
	if d.Interface.EnableProxy && d.Proxy != nil {
		err = d.Proxy.Stop()
		if err != nil {
			d.Log.Errorf("proxy server shutdown error: %s", err)
		}
		if d.Interface.Proxy.HttpsTransparentPort > 0 && d.TransparentTls != nil {
			err = d.TransparentTls.Stop()
			if err != nil {
				d.Log.Errorf("transparent HTTPS proxy server shutdown error: %s", err)
			}
		}
	}
	return err
}

func New(iface configuration.InterfaceConfig, global *configuration.DefaultConfig, logMacAddress bool, logger *log.Entry) (*Server, error) {
	var err error

	svr := Server{
		Interface:     iface,
		Log:           logger,
		LogMacAddress: logMacAddress,
	}

	// Setup HTTP service
	if iface.ShouldStartHttp() {
		logger.Debug("Creating handler HTTP")
		svr.Http = &http.Server{Handler: &svr}
		la, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", iface.Ip.String(), configuration.DefaultBindPort))
		if err != nil {
			logger.Errorf("cannot parse bind address for %s", iface.Name)
			return nil, err
		}
		svr.Listener, err = net.ListenTCP("tcp4", la)
		if err != nil {
			logger.Errorf("cannot bind address for %s", iface.Name)
			return nil, err
		}

		// Setup WPAD service
		if iface.EnableWpad {
			buf := new(bytes.Buffer)
			err = wpadFile.Execute(buf, iface)
			if err != nil {
				logger.Errorf("cannot execute WPAD template; %s", err)
				return nil, err
			}
			svr.WpadFile = buf.String()
		}

		// Setup reverse proxy service
		svr.ReverseProxies = make(map[string]reverseProxy, len(iface.ReverseProxies))
		for name, config := range iface.ReverseProxies {
			targetUrl, _ := url.Parse(fmt.Sprintf("http://%s:%d/", config.PeerIp.String(), config.PeerPort))
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

			methods := make(map[string]bool, len(config.AllowedMethods))
			for _, method := range config.AllowedMethods {
				methods[method] = true
			}
			rProxy := reverseProxy{
				Proxy:   *proxy,
				Methods: methods,
			}
			svr.ReverseProxies[name] = rProxy
		}
	}

	// Setup proxy service
	if iface.EnableProxy {
		svr.Proxy, err = NewProxy(iface, global, svr.LogMacAddress, logger)
		if err != nil {
			logger.Errorf("cannot create HTTP Proxy server: %s", err)
			return nil, err
		}
		if iface.Proxy.HttpsTransparentPort > 0 {
			svr.TransparentTls, err = NewTransparentTlsProxy(iface, svr.Proxy.Proxy, logMacAddress, logger)
			if err != nil {
				logger.Errorf("cannot create HTTPS Proxy server: %s", err)
				return nil, err
			}
		}
	}
	return &svr, nil
}
