package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
)

var (
	ErrInvalidId = errors.New("Invalid upload ID")
)

type NewUploadArgs struct{}

type NewUploadResponse struct {
	Ulid uint64
}

type StatusArgs struct {
	Ulid uint64
}

type StatusResponse struct {
	Status Progress
}

type UpdatesArgs struct {
	LastId int64
}

type UpdatesResponse struct {
	LastId  int64
	Updates string
}

type UploadService struct{}

func (u *UploadService) NewUpload(r *http.Request, req *NewUploadArgs, resp *NewUploadResponse) error {
	ulid := makeUpload()
	resp.Ulid = ulid
	log.Println("New upload requested:", ulid)
	return nil
}

func (u *UploadService) Status(r *http.Request, req *StatusArgs, resp *StatusResponse) error {
	uploadsMutex.RLock()
	ul, ok := uploads[req.Ulid]
	uploadsMutex.RUnlock()
	if !ok {
		return ErrInvalidId
	}
	ul.mutex.RLock()
	if ul.destroyed {
		ul.mutex.RUnlock()
		return ErrInvalidId
	}
	resp.Status = ul.p
	ul.mutex.RUnlock()
	return nil
}

func (u *UploadService) Updates(r *http.Request, req *UpdatesArgs, resp *UpdatesResponse) error {
	files, err := GetFilesSinceId(req.LastId)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		resp.LastId = req.LastId
		return nil
	}
	sb := new(bytes.Buffer)
	templates.ExecuteTemplate(sb, "filelist.html", files)
	resp.LastId = files[len(files)-1].id
	resp.Updates = sb.String()
	return nil
}
