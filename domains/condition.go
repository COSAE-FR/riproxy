package domains

import (
	"github.com/elazarl/goproxy"
	"net/http"
)

func DstHostIsIn(list DomainTree) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		return list.Get(req.URL.Host)
	}
}
