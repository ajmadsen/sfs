package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"path"
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
}

func (f *file) Dir() string {
	return path.Clean(filesDir)
}

func (f *file) FullName() string {
	return path.Base(string(f.name))
}

func (f *file) Name() string {
	// parts := strings.SplitN(f.FullName(), "_", 2)
	// if len(parts) != 2 {
	// 	panic("Invalid filename")
	// }
	// return parts[1]
	return f.FullName()
}

func (f *file) URL() string {
	return path.Join(filesPath, fmt.Sprint(f.id))
}

func (f *file) Path() string {
	return path.Join(f.Dir(), fmt.Sprint(f.id))
}

func (f *file) String() string {
	return f.Name()
}

func SanitizeName(f string) file {
	sanitized := nameSanitizer.ReplaceAllString(path.Base(f), ".")
	return file{name: sanitized}
}

func addFile(f *file) error {
	fname := f.FullName()
	res, err := filedb.Exec("INSERT INTO files (path, date_posted) VALUES (?, ?)", fname, time.Now().Unix())
	if err == nil {
		f.id, _ = res.LastInsertId()
	}
	return err
}

func getFile(id int64) (file, error) {
	res := filedb.QueryRow("SELECT id, path FROM files WHERE id = ?", id)
	var filename string
	var idd int64
	err := res.Scan(&idd, &filename)
	if err != nil {
		return file{}, err
	}
	return file{filename, idd}, nil
}

func getFilesSince(t time.Time) ([]file, error) {
	rows, err := filedb.Query("SELECT id, path FROM files WHERE date_posted > ? ORDER BY date_posted DESC", t.Unix())
	if err != nil {
		return nil, err
	}
	var files []file
	for rows.Next() {
		var id int64
		var path string
		err = rows.Scan(&id, &path)
		if err != nil {
			rows.Close()
			return nil, err
		}
		files = append(files, file{path, id})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return files, nil
}

func init() {
	var err error
	filedb, err = sql.Open("sqlite3", databasePath)
	if err != nil {
		panic(err)
	}
	_, err = filedb.Exec("CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY ASC, path, date_posted)")
	if err != nil {
		panic(err)
	}
}
