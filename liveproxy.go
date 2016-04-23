package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/buger/goterm"
	"gopkg.in/elazarl/goproxy.v1"
)

type Session struct {
	Time time.Time
	Ctx  *goproxy.ProxyCtx
}

func (this *Session) PrintTo(table io.Writer) {

	if this.Ctx.Resp == nil {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			"-",
			limitStrlen(this.Ctx.Req.URL.Host, 25),
			limitStrlen(this.Ctx.Req.URL.Path, 40),
			"-",
			time.Since(this.Time).String(),
		)

		return
	}

	fmt.Fprintf(table, "%s\t%s\t%s\t%d\t%s\n",
		this.coloredStatus(this.Ctx.Resp.StatusCode),
		limitStrlen(this.Ctx.Req.URL.Host, 25),
		limitStrlen(this.Ctx.Req.URL.Path, 40),
		this.Ctx.Resp.ContentLength,
		time.Since(this.Time).String(),
	)
}

func (this *Session) coloredStatus(statusCode int) string {
	status := strconv.Itoa(statusCode)

	return status

	if statusCode < 200 {
		return goterm.Color(status, goterm.CYAN)
	}

	if statusCode >= 200 && statusCode < 300 {
		return goterm.Color(status, goterm.GREEN)
	}

	if statusCode >= 300 && statusCode < 400 {
		return goterm.Color(status, goterm.YELLOW)
	}

	if statusCode >= 400 && statusCode < 500 {
		return goterm.Color(status, goterm.RED)
	}

	return goterm.Color(status, goterm.MAGENTA)
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

	RedrawScreen()

	go func() {
		for {
			ctx := <-report

			if ctx.Resp != nil {
				session := act_sessions[ctx.Session]
				session.Ctx = ctx

				hst_sessions[ctx.Session] = session
				delete(act_sessions, ctx.Session)
			} else {
				act_sessions[ctx.Session] = Session{Time: time.Now(), Ctx: ctx}
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

	height := goterm.Height() - 5

	goterm.Println(goterm.Bold("Active connections"))
	table := goterm.NewTable(5, 4, 2, ' ', 0)
	fmt.Fprintf(table, "STATUS\tHOST\tPATH\tSIZE\tAGE\n")

	for _, ctx := range act_sessions {
		if height > goterm.CurrentHeight() {
			ctx.PrintTo(table)
		}
	}

	for _, ctx := range hst_sessions {
		if height > goterm.CurrentHeight() {
			ctx.PrintTo(table)
		}
	}

	goterm.Println(table)
	goterm.MoveCursor(0, 0)
	goterm.Flush()
}

func limitStrlen(in string, limit int) string {
	if len(in) > limit {
		return in[0:limit]
	}

	return in
}
