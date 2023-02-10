package zigcentral

import (
	"database/sql"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type Package struct {
	ID  int64
	URL string
}

type PackageInfo struct {
	Sha    string
	Name   string
	Readme string
	Hash   string
}

func GetPackages(db *sql.DB) []Package {
	pkgs := make([]Package, 0)
	rows, _ := db.Query("SELECT id, url FROM packages")
	defer rows.Close()
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

func (p *Package) GetInfo(db *sql.DB) *PackageInfo {
	info := &PackageInfo{}
	u, _ := url.Parse(p.URL)
	out, err := exec.Command("git", "ls-remote", p.URL, "HEAD").Output()
	if err != nil {
		return nil
	}
	sha := string(out[0:40])
	info.Name = strings.Split(u.Path, "/")[2]
	info.Sha = sha

	// Hash can be empty, compute hash
	row := db.QueryRow("SELECT hash FROM package_hashes WHERE package_id = ? and sha_commit = ?", p.ID, info.Sha)
	err = row.Scan(&info.Hash)
	if err != nil || info.Hash == "" {
		go func() {
			res, err := http.Get(p.URL + "/archive/" + info.Sha + ".tar.gz")
			if err != nil {
				return
			}
			defer res.Body.Close()
			path, err := ExtractTarGz(res.Body)
			defer os.RemoveAll(path)
			if err != nil {
				return
			}
			hash := ComputeHash(path)
			db.Exec("INSERT INTO package_hashes(package_id, sha_commit, hash) VALUES(?, ?, ?)", p.ID, info.Sha, hash)
		}()
	}

	// README is optional
	res, err := http.Get(p.URL + "/raw/HEAD/README.md")
	if err != nil {
		return info
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		b, _ := io.ReadAll(res.Body)
		info.Readme = string(b)
	}
	return info
}
