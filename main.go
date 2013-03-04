package main

import (
	"log"
	"net/http"
)

const (
	statusPath  = "/status/"
	staticPath  = "/static/"
	updatesPath = "/updates/"
	newFilePath = "/new_file"
	uploadPath  = "/upload"
)

func main() {
	http.HandleFunc(newFilePath, HandleNewUpload)
	http.HandleFunc(uploadPath, HandleUpload)
	http.Handle(staticPath, http.StripPrefix(staticPath, http.FileServer(http.Dir(staticPath[1:]))))
	http.Handle(filesPath, http.StripPrefix(filesPath, http.HandlerFunc(HandleFile)))
	http.Handle(statusPath, http.StripPrefix(statusPath, http.HandlerFunc(HandleProgress)))
	http.Handle(updatesPath, http.StripPrefix(updatesPath, http.HandlerFunc(HandleUpdates)))
	http.HandleFunc("/", HandleMain)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err.Error())
	}
}
