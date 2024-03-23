package main

import (
	"context"
	"flag"
	"github.com/gin-gonic/gin"
	pkgHttp "github.com/smalls0098/caller/pkg/app/server/http"
	"log"
	"net/http"
	"time"

	coreCaller "github.com/smalls0098/caller"
	pkgApp "github.com/smalls0098/caller/pkg/app"
)

var (
	pwd string
	p   int
)

func init() {
	flag.IntVar(&p, "p", 13802, "server port, default: 13802")
	flag.StringVar(&pwd, "pwd", "", "check password")
}

func main() {
	// 执行命令行
	flag.Parse()

	gin.SetMode(gin.ReleaseMode)
	s := pkgHttp.NewServer(
		gin.New(),
		pkgHttp.WithServerHost("0.0.0.0"),
		pkgHttp.WithServerPort(p),
		pkgHttp.WithServerTimeout(20*time.Second),
	)
	s.Use(gin.Recovery())
	s.NoRoute(noHandle)
	s.NoMethod(noHandle)
	s.GET("/", index)
	s.POST("/call", caller)

	app := pkgApp.New(
		pkgApp.WithServer(s),
		pkgApp.WithName("caller"),
	)
	log.Printf("running: [http://127.0.0.1:%d]", p)
	if err := app.Run(context.Background()); err != nil {
		panic(err)
	}
}

func index(ctx *gin.Context) {
	ctx.String(http.StatusOK, "nw smalls caller\nok")
}

func noHandle(ctx *gin.Context) {
	ctx.String(http.StatusNotFound, "not found")
}

func caller(ctx *gin.Context) {
	if len(pwd) > 0 {
		pass := ctx.Query("pwd")
		if pass != pwd {
			ctx.String(http.StatusUnauthorized, "unauthorized")
			return
		}
	}
	coreCaller.Server(ctx.Writer, ctx.Request)
}
