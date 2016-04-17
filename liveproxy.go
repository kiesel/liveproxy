package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/buger/goterm"
	"gopkg.in/elazarl/goproxy.v1"
)

type Session struct {
	Ctx *goproxy.ProxyCtx
}

var (
	addr string

	act_sessions map[int64]Session
	hst_sessions map[int64]Session
)

func init() {
	flag.StringVar(&addr, "add", ":8080", "Port number")
	act_sessions = make(map[int64]Session, 100)
	hst_sessions = make(map[int64]Session, 100)
}

func main() {
	fmt.Println("Starting proxy ...")

	var goproxyCaErr error
	CA_CERT, err := ioutil.ReadFile("cert.pem")
	if err != nil {
		panic(err)
	}

	CA_KEY, err := ioutil.ReadFile("key.pem")
	if err != nil {
		panic(err)
	}

	goproxy.GoproxyCa, goproxyCaErr = tls.X509KeyPair(CA_CERT, CA_KEY)

	if goproxyCaErr != nil {
		panic(goproxyCaErr)
	}

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

				hst_sessions[ctx.Session] = Session{Ctx: ctx}
				delete(act_sessions, ctx.Session)
			} else {
				act_sessions[ctx.Session] = Session{Ctx: ctx}

				fmt.Println("Opening session", ctx.Session, "for", ctx.Req.URL)
			}

			RedrawScreen()
		}
	}()

	fmt.Println(http.ListenAndServe(addr, proxy))
}

func RedrawScreen() {
	goterm.Clear()
	goterm.MoveCursor(0, 0)
	goterm.Flush()

	fmt.Println(goterm.Bold("Active connections"))
	for _, ctx := range act_sessions {
		fmt.Printf(" * [%v] @ %s\n", ctx.Ctx.Session, limitStrlen(ctx.Ctx.Req.URL.String(), 40))
	}

	fmt.Println(goterm.Bold("Closed connections"))

	for _, ctx := range hst_sessions {
		fmt.Printf("   [%v] %d @ %s\n", ctx.Ctx.Session, ctx.Ctx.Resp.StatusCode, limitStrlen(ctx.Ctx.Req.URL.String(), 40))
	}
}

func limitStrlen(in string, limit int) string {
	if len(in) > limit {
		return in[0:limit]
	}

	return in
}
