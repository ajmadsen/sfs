package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	filesPath = "/files/"
	filesDir  = "/Volumes/Storage/files"
	maxSize   = 20 * 1024 * 1024
)

var (
	nameSanitizer = regexp.MustCompile(`[^\w-.]`)
)

func HandleNewUpload(w http.ResponseWriter, r *http.Request) {
	id := makeUpload()
	respstruct := struct {
		Status string
		Ulid   string
	}{"success", strconv.FormatUint(id, 16)}
	var response []byte
	response, err := json.Marshal(respstruct)
	if err != nil {
		http.Error(w, "{\"Status\": \"error\", \"Etring\": \""+err.Error()+"\"}", http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.URL.Path, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fn, err := getFile(int64(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(fn.Path())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Add("Content-Disposition", "attachment; filename=\""+fn.Name()+"\"")
	http.ServeContent(w, r, "", time.Time{}, f)
}

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	ulid := r.URL.Query().Get("ul")
	if ulid == "" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseUint(ulid, 16, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusForbidden)
		return
	}
	if r.ContentLength > maxSize {
		http.Error(w, "File too large", http.StatusNotAcceptable)
		return
	}
	pr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, err := pr.NextPart()
	for ; p.FormName() != "file" && err == nil; p, err = pr.NextPart() {
	}
	if err != nil {
		http.Error(w, "Invalid request", http.StatusNotAcceptable)
		return
	}
	filest := SanitizeName(p.FileName())
	f, err := os.Create(filest.Path())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uploadsMutex.RLock()
	ul := uploads[id]
	uploadsMutex.RUnlock()
	ul.size = r.ContentLength
	buf := make([]byte, 1024*1024)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			wr, err2 := f.Write(buf[0:n])
			if err2 != nil {
				http.Error(w, err2.Error(), http.StatusInternalServerError)
				f.Close()
				os.Remove(filest.Path())
				return
			}
			ul.mutex.Lock()
			ul.uploaded += int64(wr)
			ul.mutex.Unlock()
		}
		if err == io.EOF {
			ul.mutex.Lock()
			ul.completed = true
			ul.mutex.Unlock()
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			f.Close()
			os.Remove(filest.Path())
			return
		}
	}
	f.Close()
	p.Close()
	err = addFile(&filest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		os.Remove(filest.Path())
	}
}

func HandleProgress(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Path
	id, err := strconv.ParseUint(sid, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uploadsMutex.RLock()
	ul, ok := uploads[id]
	uploadsMutex.RUnlock()
	if !ok {
		http.Error(w, "No such ID", http.StatusNotAcceptable)
		return
	}
	ul.mutex.RLock()
	total := ul.size
	uled := ul.uploaded
	completed := ul.completed
	ul.mutex.RUnlock()
	var resp []byte
	respstruct := struct {
		Status    string
		Total     int64
		Uled      int64
		Completed bool
	}{"success", total, uled, completed}
	resp, err = json.Marshal(respstruct)
	if err != nil {
		http.Error(w, "{\"Status\": \"error\", \"Error\": \""+err.Error()+"\"}", http.StatusInternalServerError)
		return
	}
	if completed {
		uploadsMutex.Lock()
		delete(uploads, ul.id)
		uploadsMutex.Lock()
	}
	w.Write(resp)
}

func HandleUpdates(w http.ResponseWriter, r *http.Request) {
	stime := r.URL.Path
	isec, err := strconv.ParseInt(stime, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}
	t := time.Unix(isec, 0)
	files, err := getFilesSince(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = files // TODO: Everything
}
