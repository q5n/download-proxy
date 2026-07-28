package main


import (
	"log"
	"net/http"

	"github.com/q5n/download-proxy/internal/config"
	"github.com/q5n/download-proxy/internal/proxy"
)


func main(){

	cfg,err:=config.Load(
		"config.yaml",
	)

	if err!=nil{
		panic(err)
	}

	p:=proxy.New(cfg)

	http.HandleFunc(
		"/download",
		p.Handler,
	)

	log.Println(
		"listen",
		cfg.Listen,
	)

	err=http.ListenAndServe(
		cfg.Listen,
		nil,
	)

	if err!=nil{
		panic(err)
	}
}