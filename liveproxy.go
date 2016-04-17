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
	// proxy.Verbose = true

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			return goproxy.AlwaysMitm(host, ctx)
		})

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*$"))).
		DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			fmt.Println("Open", r.URL)
			fmt.Println("Session", ctx.Session)
			return r, nil
		})

	proxy.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		fmt.Println("Close", ctx.Req.URL)
		fmt.Println("Session", ctx.Session)
		return r
	})

	fmt.Println(http.ListenAndServe(addr, proxy))
}
