package server

import (
	"context"
	"fmt"
	"github.com/COSAE-FR/riproxy/configuration"
	"github.com/COSAE-FR/riproxy/domains"
	"github.com/COSAE-FR/riproxy/utils"
	"github.com/elazarl/goproxy"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func DstHostIsIP() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		hostParts := strings.Split(ctx.Req.Host, ":")
		destHost := ctx.Req.Host
		if len(hostParts) == 2 {
			destHost = hostParts[0]
		}
		return net.ParseIP(destHost) != nil
	}
}

func DstPortIsblocked(configuration configuration.InterfaceProxyConfig) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		hostParts := strings.Split(ctx.Req.Host, ":")
		destPort := "80"
		if len(hostParts) == 2 {
			destPort = hostParts[1]
		}
		if destPort == "80" { // Always allow port 80
			return false
		}
		port, err := strconv.ParseUint(destPort, 10, 16)
		if err != nil { // Block if cannot parse port
			return true
		}
		if port == 80 { // Always allow port 80
			return false
		}
		if !configuration.AllowHighPorts && port > 1024 {
			return true
		}
		if !configuration.AllowLowPorts && port <= 1024 {
			return true
		}
		return false
	}
}

func MethodIsBlocked(allowed map[string]bool) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		return !allowed[ctx.Req.Method]
	}
}

func IpIsBlocked(blockList []net.IP) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		hostParts := strings.Split(ctx.Req.Host, ":")
		destIP, err := net.ResolveIPAddr("ip", hostParts[0])
		if err == nil {
			return connectTestDestIp(destIP.IP, blockList)
		}
		return false
	}
}

func addBlockList(proxy *goproxy.ProxyHttpServer, message string, list domains.DomainTree, logger *log.Entry) *goproxy.ProxyHttpServer {
	proxy.OnRequest(domains.DstHostIsIn(list)).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		prepareRequestLogger(logger, ctx).Error(message)
		return req, goproxy.NewResponse(req,
			goproxy.ContentTypeText, http.StatusForbidden,
			message)
	})
	return proxy
}

func connectTestPort(portString string, configuration configuration.InterfaceProxyConfig) bool {
	if portString == "443" { // Always allow port 443 for CONNECT
		return true
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil { // Block if cannot parse port
		return false
	}
	if port == 443 { // Always allow port 443 for CONNECT
		return true
	}
	if !configuration.AllowHighPorts && port > 1024 {
		return false
	}
	if !configuration.AllowLowPorts && port <= 1024 {
		return false
	}
	return true
}

func connectTestDestIp(destIp net.IP, blockList []net.IP) bool {
	for _, blocked := range blockList {
		if destIp.Equal(blocked) {
			return true
		}
	}
	return false
}

var logHeaders = map[string]string{
	"User-Agent":   "user_agent",
	"Referer":      "referrer",
	"Content-Type": "content_type",
}

func prepareRequestLogger(logger *log.Entry, ctx *goproxy.ProxyCtx) *log.Entry {
	ip, port := utils.GetConnection(ctx.Req.RemoteAddr)
	requestLogger := logger.WithFields(log.Fields{
		"src":      ip.String(),
		"src_port": port,
		"method":   ctx.Req.Method,
		"url":      ctx.Req.URL.String(),
		"action":   "pass",
		"bytes_in": ctx.Req.ContentLength,
	})
	for header, logField := range logHeaders {
		field := ctx.Req.Header.Get(header)
		if len(field) > 0 {
			requestLogger = requestLogger.WithField(logField, field)
		}
	}
	if ctx.Resp != nil {
		requestLogger = requestLogger.WithFields(log.Fields{
			"status":    ctx.Resp.StatusCode,
			"bytes_out": ctx.Resp.ContentLength,
		})
	}
	return requestLogger
}

type ProxyServer struct {
	Interface configuration.InterfaceConfig
	Global    *configuration.DefaultConfig
	Listener  *net.TCPListener
	Http      *http.Server
	Log       *log.Entry
	Proxy     *goproxy.ProxyHttpServer
}

func (p ProxyServer) Start() error {
	p.Log.Debug("starting HTTP Proxy daemon")
	go func() {
		err := p.Http.Serve(p.Listener)
		if err != http.ErrServerClosed {
			p.Log.Debugf("proxy server stopped with error: %s", err)
		}
	}()
	return nil
}

func (p ProxyServer) Stop() error {
	p.Log.Debugf("stopping HTTP Proxy daemon")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Http.Shutdown(ctx); err != nil {
		return err
	} else {
		p.Log.Debug("gracefully stopped")
	}
	return nil
}

func NewProxy(iface configuration.InterfaceConfig, global *configuration.DefaultConfig, logger *log.Entry) (*ProxyServer, error) {
	proxyLogger := logger.WithFields(log.Fields{
		"component": "proxy",
		"ip":        iface.Ip.String(),
		"port":      iface.Proxy.Port,
	})
	proxy := goproxy.NewProxyHttpServer()

	// Block if destination is a local service
	if iface.Proxy.BlockLocalServices {
		proxy.OnRequest(IpIsBlocked(iface.Proxy.LocalIps)).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			prepareRequestLogger(logger, ctx).Error("Blocked: destination is not allowed: local service")
			return req, goproxy.NewResponse(req,
				goproxy.ContentTypeText, http.StatusForbidden,
				"Blocked: destination is not allowed")
		})
	}

	// Block if method is not allowed
	allowedMethods := make(map[string]bool, len(iface.Proxy.AllowedMethods))
	for _, method := range iface.Proxy.AllowedMethods {
		allowedMethods[method] = true
	}
	proxy.OnRequest(MethodIsBlocked(allowedMethods)).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		prepareRequestLogger(logger, ctx).Errorf("Blocked: method not allowed: %+v %+v", allowedMethods[ctx.Req.Method], allowedMethods)
		return req, goproxy.NewResponse(req,
			goproxy.ContentTypeText, http.StatusForbidden,
			fmt.Sprintf("Blocked: method %s not allowed", ctx.Req.Method))
	})

	// Block if dest port is not allowed
	proxy.OnRequest(DstPortIsblocked(iface.Proxy)).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		prepareRequestLogger(logger, ctx).Error("Blocked by host port policy")
		return req, goproxy.NewResponse(req,
			goproxy.ContentTypeText, http.StatusForbidden,
			"Blocked by host port policy")
	})
	// Block host IPs if configured
	if iface.Proxy.BlockIPs {
		proxy.OnRequest(DstHostIsIP()).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			prepareRequestLogger(logger, ctx).Error("Blocked by host policy")
			return req, goproxy.NewResponse(req,
				goproxy.ContentTypeText, http.StatusForbidden,
				"Blocked by host policy")
		})
	}

	// Add interface domain block list
	if iface.Proxy.BlockList != nil {
		proxy = addBlockList(proxy, "Blocked by interface policy", iface.Proxy.BlockList, proxyLogger)
	}

	// Add global domain block list
	if global != nil && global.Proxy.BlockList != nil {
		proxy = addBlockList(proxy, "Blocked by global policy", global.Proxy.BlockList, proxyLogger)
	}
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		requestLogger := prepareRequestLogger(proxyLogger, ctx)
		if ctx.Resp == nil {
			requestLogger.WithField("action", "error").Error("Proxy request: no response")
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
			"src":        ip.String(),
			"src_port":   port,
			"method":     ctx.Req.Method,
			"url":        url,
			"dest":       destHost,
			"dest_port":  destPort,
			"user_agent": ctx.Req.Header.Get("User-Agent"),
			"action":     "tunnel",
		})
		if iface.Proxy.BlockLocalServices {
			destIP, err := net.ResolveIPAddr("ip", destHost)
			if err == nil {
				if connectTestDestIp(destIP.IP, iface.Proxy.LocalIps) {
					requestLogger.WithField("action", "block").Error("Blocked: destination is not allowed: local service")
					return goproxy.RejectConnect, host
				}
			}
		}
		if !allowedMethods[ctx.Req.Method] {
			requestLogger.WithField("action", "block").Error("Connect method blocked by policy")
			return goproxy.RejectConnect, host
		}
		if iface.Proxy.BlockIPs {
			if net.ParseIP(destHost) != nil {
				requestLogger.WithField("action", "block").Errorf("Connect to IP host %s not allowed", destHost)
				return goproxy.RejectConnect, host
			}
		}
		if !connectTestPort(destPort, iface.Proxy) {
			requestLogger.WithField("action", "block").Errorf("Connect port %s not allowed", destPort)
			return goproxy.RejectConnect, host
		}
		requestLogger.Info("Connect request")
		return goproxy.OkConnect, host
	})
	proxy.Logger = proxyLogger
	la, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", iface.Ip, iface.Proxy.Port))
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
		Global:    global,
		Log:       proxyLogger,
		Proxy:     proxy,
		Listener:  listener,
		Http:      &http.Server{Handler: proxy},
	}
	return &proxyServer, nil
}
