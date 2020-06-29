package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/mod/semver"
)

type db struct {
	manifest Manifest
	entries  map[int]dbEntry
	nextID   int
}

type dbEntry struct {
	versions map[string]dbPackage
}

type dbPackage struct {
	manifest ManifestEntry
	pkg      Package
}

func buildDb(manifest Manifest) (db, error) {
	result := db{manifest: manifest, entries: map[int]dbEntry{}, nextID: 1}
	for _, entry := range manifest {
		result.buildManifest(entry)
	}
	return result, nil
}

func (db *db) buildManifest(manifest ManifestEntry) error {
	if db.nextID <= manifest.ModID {
		db.nextID = manifest.ModID + 1
	}

	reader, err := zip.OpenReader("data/" + manifest.Hash + ".zip")
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name == manifest.Path {
			r, err := file.Open()
			defer r.Close()
			if err != nil {
				return err
			}

			var pkg Package
			err = json.NewDecoder(r).Decode(&pkg)
			if err != nil {
				return err
			}

			db.addDbPackage(dbPackage{
				manifest: manifest,
				pkg:      pkg,
			})

			return nil
		}
	}

	return os.ErrNotExist
}

func (db db) addDbPackage(pkg dbPackage) {
	entry, ok := db.entries[pkg.manifest.ModID]
	if !ok {
		entry = dbEntry{
			versions: map[string]dbPackage{},
		}
		db.entries[pkg.manifest.ModID] = entry
	}

	entry.versions[pkg.pkg.Version] = pkg
}

func (db *db) newEntryAndID(url string, path string) error {
	id := db.nextID
	db.nextID++
	return db.newEntry(id, url, path)
}

func (db *db) newEntry(id int, url string, path string) error {
	data, err := download(url)
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write(data)
	hash := hex.EncodeToString(h.Sum(nil))

	err = ioutil.WriteFile("data/"+hash+".zip", data, 0666)
	if err != nil {
		return err
	}

	manifest := ManifestEntry{
		ModID:    id,
		URL:      url,
		Hash:     hash,
		Path:     path,
		Approved: false,
	}

	err = db.manifest.Add(manifest)
	if err != nil {
		return err
	}

	return db.buildManifest(manifest)
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (dbe dbEntry) newest() dbPackage {
	result := dbPackage{}
	newest := "v0.0.0"
	for ver, pkg := range dbe.versions {
		ver = "v" + ver
		cp := semver.Compare(newest, ver)
		if cp < 0 {
			result = pkg
			newest = ver
		}
	}
	return result
}

func (dbp dbPackage) GetName() string {
	if dbp.pkg.DisplayName != "" {
		return dbp.pkg.DisplayName
	}
	return dbp.pkg.Name
}
