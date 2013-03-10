package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"
)

const (
	filesPath  = "/files/"
	filesDir   = "./files"
	tempPrefix = "sfsUpload"
	maxSize    = 20 * 1024 * 1024 * 1024
)

var (
	nameSanitizer = regexp.MustCompile(`[^\w-. ]`)
	templates     = template.Must(template.ParseFiles("tmpl/filelist.html", "tmpl/index.html"))
	tempDir       = path.Join(filesDir, "temp/")
)

type mainpage struct {
	Files []*file
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.URL.Path, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fn, err := GetFileById(int64(id))
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
	log.Println("File", fn.Name(), "requested from", r.RemoteAddr)
	http.ServeContent(w, r, "", time.Time{}, f)
}

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	var id uint64
	ulid := r.URL.Query().Get("ul")
	if ulid != "" {
		var err error
		id, err = strconv.ParseUint(ulid, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusForbidden)
			return
		}
	} else {
		id = makeUpload()
	}
	if r.ContentLength > maxSize {
		http.Error(w, "File too large", http.StatusForbidden)
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
		http.Error(w, "Invalid request", http.StatusForbidden)
		return
	}
	defer p.Close()
	filest, err := PrepareFile(p.FileName())
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := filest.TempFile()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uploadsMutex.RLock()
	ul, ok := uploads[id]
	uploadsMutex.RUnlock()
	if !ok {
		http.Error(w, ErrInvalidId.Error(), http.StatusInternalServerError)
		return
	}
	ul.mutex.Lock()
	if ul.destroyed {
		ul.mutex.Unlock()
		http.Error(w, "Invalid upload id", http.StatusForbidden)
		return
	}
	ul.p.Total = r.ContentLength
	ul.p.Started = true
	ul.mutex.Unlock()
	buf := make([]byte, 1024*1024)
	log.Print("Upload started from:", r.RemoteAddr)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			wr, err2 := f.Write(buf[0:n])
			if err2 != nil {
				http.Error(w, err2.Error(), http.StatusInternalServerError)
				filest.Cancel()
				ul.delete()
				return
			}
			ul.mutex.Lock()
			ul.p.Uploaded += int64(wr)
			ul.mutex.Unlock()
		}
		if err == io.EOF {
			ul.mutex.Lock()
			ul.p.Completed = true
			ul.mutex.Unlock()
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			filest.Cancel()
			ul.delete()
			return
		}
	}
	err = filest.Commit()
	if err != nil {
		filest.Cancel()
		ul.delete()
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	log.Println("Upload complete")
}

func HandleMain(w http.ResponseWriter, r *http.Request) {
	log.Println("Main page requested from", r.RemoteAddr)
	files, err := GetFilesSinceTime(time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mp := mainpage{files}
	err = templates.ExecuteTemplate(w, "index.html", mp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
