package main

import (
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"zigcentral"

	_ "modernc.org/sqlite"
)

func migrate(db *sql.DB) {
	_, err := db.Exec("create table packages(id integer primary key autoincrement, url text not null);")
	if err != nil {
		log.Println(err)
	}
}

func main() {
	db, err := sql.Open("sqlite", "database.db")
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	migrated := true

	exists := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'packages';")
	if err != nil {
		log.Fatalln(err)
	}
	var tablename string
	err = exists.Scan(&tablename)
	if err != nil {
		migrated = false
	}

	if !migrated {
		migrate(db)
	}

	h := zigcentral.NewHandlers(db)
	static, _ := fs.Sub(zigcentral.Files, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	http.HandleFunc("/", h.HomePage)
	http.HandleFunc("/post", h.PostPage)
	http.HandleFunc("/pkg/", h.PackagePage)
	http.HandleFunc("/action/post", h.PostAction)
	http.ListenAndServe(":8080", nil)
}
