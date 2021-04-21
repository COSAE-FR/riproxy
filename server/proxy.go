package server

import (
	"context"
	"fmt"
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/elazarl/goproxy"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"time"
)

type ProxyServer struct {
	Interface configuration.InterfaceConfig
	Listener  *net.TCPListener
	Http      *http.Server
	Log       *log.Entry
	Proxy     *goproxy.ProxyHttpServer
}

func (p ProxyServer) Start() error {
	p.Log.Debug("starting HTTP Proxy daemon")
	go p.Http.Serve(p.Listener)
	return nil
}

func (p ProxyServer) Stop() error {
	var err error
	p.Log.Debugf("stopping HTTP Proxy daemon")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = p.Http.Shutdown(ctx); err != nil {
		p.Log.Errorf("shutdown error: %v", err)
	} else {
		p.Log.Debug("gracefully stopped")
	}
	return err
}

func NewProxy(iface configuration.InterfaceConfig, logger *log.Entry) (*ProxyServer, error) {
	proxyLogger := logger.WithFields(log.Fields{
		"component": "proxy",
		"ip":        iface.ProxyIP.String(),
		"port":      iface.ProxyPort,
	})
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		ip, port := utils.GetConnection(ctx.Req.RemoteAddr)
		requestLogger := proxyLogger.WithFields(log.Fields{
			"src":             ip.String(),
			"src_port":        port,
			"http_method":     ctx.Req.Method,
			"uri_path":        ctx.Req.URL.Path,
			"url":             ctx.Req.URL.String(),
			"http_user_agent": ctx.Req.Header.Get("User-Agent"),
			"http_referrer":   ctx.Req.Header.Get("Referer"),
			"action":          "pass",
		})
		if ctx.Resp != nil {
			requestLogger = requestLogger.WithFields(log.Fields{
				"status": ctx.Resp.StatusCode,
			})
		} else {
			requestLogger.WithFields(log.Fields{
				"action": "error",
			}).Error("Proxy request: no response")
			return resp
		}
		requestLogger.Info("Proxy request")
		return resp
	})
	proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		ip, port := utils.GetConnection(ctx.Req.RemoteAddr)
		hostParts := strings.Split(host, ":")
		destPort := "443"
		destHost := host
		if len(hostParts) == 2 {
			destHost = hostParts[0]
			destPort = hostParts[1]
		}
		url := ctx.Req.URL.String()
		if len(url) > 0 && !strings.HasPrefix(url, "https") {
			url = fmt.Sprintf("https:%s", url)
		}
		requestLogger := proxyLogger.WithFields(log.Fields{
			"src":             ip.String(),
			"src_port":        port,
			"http_method":     ctx.Req.Method,
			"uri_path":        ctx.Req.URL.Path,
			"url":             url,
			"dest":            destHost,
			"dest_port":       destPort,
			"http_user_agent": ctx.Req.Header.Get("User-Agent"),
			"http_referrer":   ctx.Req.Header.Get("Referer"),
			"action":          "tunnel",
		})
		requestLogger.Info("Connect request")
		return goproxy.OkConnect, host
	})
	proxy.Logger = proxyLogger
	la, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", iface.ProxyIP, iface.ProxyPort))
	if err != nil {
		proxyLogger.Errorf("cannot parse proxy bind address for %s", iface.Name)
		return nil, err
	}
	listener, err := net.ListenTCP("tcp4", la)
	if err != nil {
		proxyLogger.Errorf("cannot bind proxy address for %s", iface.Name)
		return nil, err
	}
	proxyServer := ProxyServer{
		Interface: iface,
		Log:       proxyLogger,
		Proxy:     proxy,
		Listener:  listener,
		Http:      &http.Server{Handler: proxy},
	}
	return &proxyServer, nil
}
