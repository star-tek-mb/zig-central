package zigcentral

import (
	"database/sql"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown"
)

var (
	//go:embed resources/*.html static/**
	Files embed.FS
)

type Handlers struct {
	db        *sql.DB
	templates map[string]*template.Template
}

func NewHandlers(db *sql.DB) *Handlers {
	templates := make(map[string]*template.Template)
	tmplFiles, err := fs.ReadDir(Files, "resources")
	if err != nil {
		log.Fatalln(err)
	}

	for _, tmpl := range tmplFiles {
		if tmpl.IsDir() {
			continue
		}

		pt, err := template.ParseFS(Files, "resources/"+tmpl.Name())
		if err != nil {
			log.Fatalln(err)
		}

		templates[tmpl.Name()] = pt
	}
	return &Handlers{db: db, templates: templates}
}

func (h *Handlers) HomePage(w http.ResponseWriter, req *http.Request) {
	pkgs := GetPackages(h.db)
	tmpl := h.templates["index.html"]
	tmpl.Execute(w, map[string]any{
		"Pkgs": pkgs,
	})
}

func (h *Handlers) PackagePage(w http.ResponseWriter, req *http.Request) {
	stringID := strings.TrimPrefix(req.URL.Path, "/pkg/")
	ID, _ := strconv.ParseInt(stringID, 10, 64)
	pkg := GetPackageByID(h.db, ID)
	if pkg == nil {
		http.NotFound(w, req)
		return
	}
	info := pkg.GetInfo(h.db)
	if info == nil {
		http.NotFound(w, req)
		return
	}

	tmpl := h.templates["pkg.html"]
	tmpl.Execute(w, map[string]any{
		"Pkg":    pkg,
		"Info":   info,
		"Readme": template.HTML(markdown.ToHTML([]byte(info.Readme), nil, nil)),
	})
}

func (h *Handlers) PostPage(w http.ResponseWriter, req *http.Request) {
	tmpl := h.templates["post.html"]
	tmpl.Execute(w, nil)
}

func (h *Handlers) PostAction(w http.ResponseWriter, req *http.Request) {
	_ = req.ParseForm()
	url := req.FormValue("url")
	res, err := http.Get(url + "/raw/HEAD/build.zig")
	if err != nil {
		http.Redirect(w, req, "/post", http.StatusFound)
		return
	}
	if res.StatusCode != 200 {
		http.Redirect(w, req, "/post", http.StatusFound)
		return
	}
	defer res.Body.Close()

	exist := GetPackageByURL(h.db, url)
	if exist != nil {
		http.Redirect(w, req, "/post", http.StatusFound)
		return
	}

	p := &Package{URL: url}
	p.Save(h.db)
	go p.GetInfo(h.db)

	http.Redirect(w, req, "/", http.StatusFound)
}
