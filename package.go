package zigcentral

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Package struct {
	ID  int64
	URL string
}

type PackageInfo struct {
	Sha    string `json:"sha"`
	Name   string
	Readme string
}

func GetPackages(db *sql.DB) []Package {
	pkgs := make([]Package, 0)
	rows, _ := db.Query("SELECT id, url FROM packages")
	for rows.Next() {
		var pkg Package
		err := rows.Scan(&pkg.ID, &pkg.URL)
		if err == nil {
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs
}

func GetPackageByID(db *sql.DB, id int64) *Package {
	var pkg Package
	row := db.QueryRow("SELECT id, url FROM packages WHERE id = ?", id)
	err := row.Scan(&pkg.ID, &pkg.URL)
	if err != nil {
		return nil
	}
	return &pkg
}

func GetPackageByURL(db *sql.DB, url string) *Package {
	var pkg Package
	row := db.QueryRow("SELECT id, url FROM packages WHERE UPPER(url) = UPPER(?)", url)
	err := row.Scan(&pkg.ID, &pkg.URL)
	if err != nil {
		return nil
	}
	return &pkg
}

func (p *Package) Save(db *sql.DB) error {
	if p.ID == 0 {
		_, err := db.Exec("INSERT INTO packages(url) VALUES(?)", p.URL)
		return err
	} else {
		_, err := db.Exec("UPDATE packages SET url = ? WHERE id = ?", p.URL, p.ID)
		return err
	}
}

func (p *Package) GetInfo() *PackageInfo {
	u, _ := url.Parse(p.URL)
	githubInfoURL := "https://api.github.com/repos" + u.Path + "/commits/HEAD"
	res, err := http.Get(githubInfoURL)
	if err != nil {
		return nil
	}
	var info PackageInfo
	info.Name = strings.Split(u.Path, "/")[2]
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(b, &info)
	if err != nil {
		return nil
	}
	// README is optional
	res, err = http.Get(p.URL + "/raw/HEAD/README.md")
	if err != nil {
		return &info
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		b, _ = io.ReadAll(res.Body)
		info.Readme = string(b)
	}
	return &info
}
