package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

const (
	databasePath = "files.db"
)

var (
	filedb *sql.DB
)

type file struct {
	name string
	id   int64
	temp *os.File
	t    time.Time
	tx   *sql.Stmt
}

func (f *file) Name() string {
	return path.Base(path.Clean(f.name))
}

func (f *file) TempName() (string, error) {
	tf, err := f.TempFile()
	if err != nil {
		return "", err
	}
	return tf.Name(), nil
}

func (f *file) TempFile() (*os.File, error) {
	var err error
	if f.temp == nil {
		f.temp, err = ioutil.TempFile(tempDir, tempPrefix)
		if err != nil {
			return nil, err
		}
	}
	return f.temp, nil
}

func (f *file) UTF8Name() string {
	return strings.Replace(template.URLQueryEscaper(f.Name()), "+", "%20", -1)
}

func (f *file) ASCIIName() string {
	return nameSanitizer.ReplaceAllString(f.Name(), ".")
}

func (f *file) ID() string {
	return fmt.Sprint(f.id)
}

func (f *file) Path() string {
	return path.Join(filesDir, f.ID())
}

func (f *file) URL() string {
	return path.Join(filesPath, f.ID())
}

func (f *file) TimePosted() int64 {
	return f.t.UTC().Unix()
}

func (f *file) Commit() error {
	if f.tx != nil {
		if f.t == (time.Time{}) {
			f.t = time.Now().UTC()
		}
		res, err := f.tx.Exec(f.Name(), f.t.UTC().Unix())
		if err != nil {
			return err
		}
		err = f.tx.Close()
		f.tx = nil
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		f.id = id
	}
	if f.temp != nil {
		err := f.temp.Close()
		if err != nil {
			return err
		}
		tname, err := f.TempName()
		if err != nil {
			return err
		}
		err = os.Rename(tname, f.Path())
		if err != nil {
			return err
		}
		f.temp = nil
	}
	return nil
}

func (f *file) Cancel() error {
	err := f.c()
	if err != nil {
		log.Println(err.Error())
	}
	return err
}

func (f *file) c() error {
	if f.tx != nil {
		err := f.tx.Close()
		if err != nil {
			return err
		}
		f.tx = nil
	}
	if f.temp != nil {
		err := f.temp.Close()
		if err != nil {
			return err
		}
		tpath, err := f.TempName()
		if err != nil {
			return err
		}
		err = os.Remove(tpath)
		if err != nil {
			return err
		}
		f.temp = nil
	}
	if f.id > 0 {
		err := os.Remove(f.Path())
		if !os.IsNotExist(err) {
			return err
		}
		f.id = 0
	}
	return nil
}

func PrepareFile(name string) (f *file, err error) {
	f = new(file)
	f.name = path.Clean(name)
	f.t = time.Now()
	f.temp, err = ioutil.TempFile(tempDir, tempPrefix)
	if err != nil {
		return nil, err
	}
	f.tx, err = filedb.Prepare("INSERT INTO files (name, date_posted) VALUES (?,?)")
	if err != nil {
		return nil, err
	}
	return
}

func GetFileById(id int64) (*file, error) {
	f := new(file)
	row := filedb.QueryRow("SELECT * FROM files WHERE id = ?", id)
	var utime int64
	err := row.Scan(&f.id, &f.name, &utime)
	if err != nil {
		return nil, err
	}
	f.t = time.Unix(utime, 0)
	return f, nil
}

func GetFilesSinceId(id int64) (files []*file, err error) {
	rows, err := filedb.Query("SELECT * FROM files WHERE id > ? ORDER BY id DESC", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var utime int64
		f := new(file)
		err = rows.Scan(&f.id, &f.name, &utime)
		if err != nil {
			return nil, err
		}
		f.t = time.Unix(utime, 0)
		files = append(files, f)
	}
	return
}

func GetFilesSinceTime(t time.Time) (files []*file, err error) {
	rows, err := filedb.Query("SELECT * FROM files WHERE date_posted > ? ORDER BY date_posted DESC", t.UTC().Unix())
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var utime int64
		f := new(file)
		err = rows.Scan(&f.id, &f.name, &utime)
		if err != nil {
			return nil, err
		}
		f.t = time.Unix(utime, 0)
		files = append(files, f)
	}
	return
}

func init() {
	var err error
	filedb, err = sql.Open("sqlite3", databasePath)
	if err != nil {
		panic(err)
	}
	_, err = filedb.Exec("CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY ASC, name, date_posted)")
	if err != nil {
		panic(err)
	}
}
