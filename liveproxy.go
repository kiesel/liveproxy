package main

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"

	"gopkg.in/elazarl/goproxy.v1"
)

var (
	addr string
)

func init() {
	flag.StringVar(&addr, "add", ":8080", "Port number")
}

func main() {
	fmt.Println("Starting proxy ...")

	flag.Parse()

	proxy := goproxy.NewProxyHttpServer()
	report := make(chan *goproxy.ProxyCtx)

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			return goproxy.AlwaysMitm(host, ctx)
		})

	proxy.OnRequest().
		DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			report <- ctx
			return r, nil
		})

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		report <- ctx
		return r
	})

	go func() {
		for {
			ctx := <-report

			if ctx.Resp != nil {
				fmt.Println("Closing session", ctx.Session, "w/", ctx.Resp.StatusCode, "for", ctx.Req.URL)
			} else {
				fmt.Println("Opening session", ctx.Session, "for", ctx.Req.URL)
			}
		}
	}()

	fmt.Println(http.ListenAndServe(addr, proxy))
}
