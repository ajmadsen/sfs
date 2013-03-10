package main

import (
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"log"
	"net/http"
)

const (
	statusPath  = "/status/"
	staticPath  = "/static/"
	updatesPath = "/updates/"
	newFilePath = "/new_file"
	uploadPath  = "/upload"
	rpcPath     = "/rpc"
)

func main() {
	http.HandleFunc(uploadPath, HandleUpload)
	http.Handle(staticPath, http.StripPrefix(staticPath, http.FileServer(http.Dir(staticPath[1:]))))
	http.Handle(filesPath, http.StripPrefix(filesPath, http.HandlerFunc(HandleFile)))
	http.HandleFunc("/", HandleMain)

	pubApi := new(UploadService)
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(pubApi, "")

	http.Handle(rpcPath, s)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err.Error())
	}
}
