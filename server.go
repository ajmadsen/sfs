package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MaxSize   = 100 * 1024 * 1024 * 1024
	filesDir  = "/Volumes/Storage/files/"
	filesPath = "/files/"
)

var (
	templates             = template.Must(template.ParseFiles("tmpl/index.html"))
	users                 = make(map[uint64]user)
	usersMutex            = new(sync.RWMutex)
	errorCookieNotPresent = errors.New("Client not accepting cookies")
)

type file struct {
	Name   string
	Path   string
	Posted time.Time
}

func (f *file) String() string {
	return fmt.Sprint(f.Posted.Unix()) + "_" + f.Name
}

func (f *file) FullPath() string {
	return path.Join(f.Path, f.String())
}

type mainpage struct {
	Files []file
}

type upload struct {
	ID        uint64
	File      file
	BytesRead int64
	Size      int64
	Complete  bool
	Mutex     *sync.RWMutex
}

func newUpload(f file) upload {
	id := uint64(rand.Int63())
	return upload{ID: id, File: f, Mutex: new(sync.RWMutex)}
}

type user struct {
	Uploads []upload
}

func (u *user) Add(up upload) {
	u.Uploads = append(u.Uploads, up)
}

func (u *user) findIdx(id uint64) int {
	for i, v := range u.Uploads {
		if v.ID == id {
			return i
		}
	}
	return -1
}

func (u *user) Find(id uint64) (upload, bool) {
	i := u.findIdx(id)
	if i > 0 {
		return u.Uploads[i], true
	}
	return upload{}, false
}

func (u *user) Delete(id uint64) {
	i := u.findIdx(id)
	if i > 0 {
		u.Uploads = append(u.Uploads[i:], u.Uploads[i+1:]...)
	}
}

func readDir(dpath string) ([]file, error) {
	files, err := ioutil.ReadDir(dpath)
	if err != nil {
		return nil, err
	}
	l := len(files)
	out := make([]file, l)
	for i, f := range files {
		out[l-i-1], err = decodeFile(path.Join(dpath, f.Name()))
		if err != nil {
			log.Fatalln("readDir:", err.Error())
		}
	}
	return out, nil
}

func encodeFile(fpath string) file {
	p := path.Dir(fpath)
	n := path.Base(fpath)
	e := base64.URLEncoding.EncodeToString([]byte(n))
	t := time.Now()
	return file{n, path.Join(p, e), t}
}

func decodeFile(fpath string) (file, error) {
	p := path.Dir(fpath)
	e := path.Base(fpath)
	parts := strings.Split(e, "_")
	if len(parts) != 2 {
		return file{}, errors.New("invalid file name format")
	}
	tsec, err := strconv.ParseUint(parts[0], 10, 63)
	t := time.Unix(int64(tsec), 0)
	if err != nil {
		return file{}, err
	}
	n, err := base64.URLEncoding.DecodeString(parts[1])
	return file{string(n), path.Join(p, e), t}, nil
}

func newUser() uint64 {
	uid := uint64(rand.Int63())
	var u user
	usersMutex.Lock()
	users[uid] = u
	usersMutex.Unlock()
	return uid
}

func newUserCookie(uid uint64) *http.Cookie {
	return &http.Cookie{Name: "uid", Value: strconv.FormatUint(uid, 16), Expires: time.Now().AddDate(0, 1, 0)}
}

func uid(r *http.Request) (uint64, bool) {
	c, err := r.Cookie("uid")
	if err != nil {
		return 0, false
	}
	uid, err := strconv.ParseUint(c.Value, 16, 64)
	if err != nil {
		return 0, false
	}
	return uid, true
}

func uidOrFail(w http.ResponseWriter, r *http.Request) uint64 {
	uid, ok := uid(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		panic(errorCookieNotPresent)
	}
	return uid
}

func bail() {
	if r := recover(); r != nil {
		if r != errorCookieNotPresent {
			panic(r)
		}
	}
}

func uidOrNew(w http.ResponseWriter, r *http.Request) uint64 {
	uid, ok := uid(r)
	if !ok {
		uid := newUser()
		c := newUserCookie(uid)
		w.Header().Add("Set-Cookie", c.String())
	}
	return uid
}

func HandleUpload(w http.ResponseWriter, r *http.Request) {
	defer bail()
	log.Println("Started upload...")
	fsize := r.ContentLength
	if fsize > MaxSize {
		http.Error(w, "Filesize too large!", http.StatusInternalServerError)
		return
	}
	mpreader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p, err := mpreader.NextPart()
	var filename string
	for err == nil {
		if p.FormName() == "file" {
			filename = p.FileName()
			break
		}
		p, err = mpreader.NextPart()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filest := encodeFile(path.Join(filesPath, filename))
	outf, err := os.Create(path.Join(filesDir, path.Base(filest.FullPath())))
	if err != nil {
		log.Println("HandleUpload:", err.Error())
		return
	}
	ul := newUpload(filest)
	ul.Size = fsize
	func(w http.ResponseWriter, r *http.Request, i io.ReadCloser, o *os.File, ul upload) {
		buf := make([]byte, 1024*1024)
		for {
			n, err := i.Read(buf)
			if n > 0 {
				wr, err2 := o.Write(buf[0:n])
				if err2 != nil {
					log.Println("HandleUpload:", err2.Error())
					return
				}
				ul.Mutex.Lock()
				ul.BytesRead += int64(wr)
				ul.Mutex.Unlock()
			}
			if err == io.EOF {
				ul.Mutex.Lock()
				ul.Complete = true
				ul.Mutex.Unlock()
				break
			}
			if err != nil {
				log.Println("HandleUpload:", err.Error())
				return
			}
		}
		log.Println("File upload completed")
		i.Close()
		o.Close()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}(w, r, p, outf, ul)
}

func HandleFile(w http.ResponseWriter, r *http.Request) {
	if r.URL.String() == "" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
	file, err := os.Open(filesDir + r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	filest, err := decodeFile(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Add("Content-Disposition", "attachment; filename=\""+filest.Name+"\"")
	http.ServeContent(w, r, "", time.Time{}, file)
}

func HandleMain(w http.ResponseWriter, r *http.Request) {
	var page mainpage
	filelisting, err := readDir(filesDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page.Files = filelisting
	err = templates.ExecuteTemplate(w, "index.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	http.Handle(filesPath, http.StripPrefix(filesPath, http.HandlerFunc(HandleFile)))
	http.HandleFunc("/add_file", HandleUpload)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	http.HandleFunc("/", HandleMain)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err.Error())
	}
}
