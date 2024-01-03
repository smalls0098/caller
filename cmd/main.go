package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/smalls0098/caller"
)

var (
	p     string
	pwd   string
	debug bool
)

func init() {
	flag.StringVar(&p, "p", "13802", "server port, default: 13802")
	flag.StringVar(&pwd, "pwd", "", "check password")
	flag.BoolVar(&debug, "debug", false, "open debug mode, default: false")
}

func main() {
	// 执行命令行
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("smalls caller"))
	})
	http.HandleFunc("/call", func(w http.ResponseWriter, r *http.Request) {
		if len(pwd) > 0 {
			pass := r.URL.Query().Get("pwd")
			if pass != pwd {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte{})
				return
			}
		}
		call(w, r)
	})
	server := http.Server{Addr: "0.0.0.0:" + p}
	go func() {
		if err := server.ListenAndServe(); err != nil { // 监听处理
			log.Println("server start failed")
		}
	}()
	log.Println("running: [http://127.0.0.1:" + p + "]")

	// 通过信号量的方式停止服务，如果有一部分请求进行到一半，处理完成再关闭服务器
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	s := <-c
	log.Printf("接收信号：%s\n", s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Println("server shutdown failed")
	}
	log.Println("server exit")
}

func call(w http.ResponseWriter, r *http.Request) {
	if err := recover(); err != nil {
		log.Println(err)
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 not found"))
		return
	}
	caller.Server(w, r, debug)
}
