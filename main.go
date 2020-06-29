package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var data db

func main() {
	man, err := LoadManifest()
	if err != nil {
		panic(err)
	}
	data, err = buildDb(man)
	if err != nil {
		panic(err)
	}
	defer man.Save()

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/v1/", list)
	http.HandleFunc("/v1/info/", info)
	http.HandleFunc("/v1/latest/", latest)
	http.HandleFunc("/v1/versions/", versions)
	http.HandleFunc("/v1/download/", get)
	http.HandleFunc("/v1/register/", register)
	http.Serve(l, nil)
}

func list(rw http.ResponseWriter, req *http.Request) {
	result := map[string]int{}
	for id, entry := range data.entries {
		result[entry.newest().GetName()] = id
	}
	json.NewEncoder(rw).Encode(result)
}

func info(rw http.ResponseWriter, req *http.Request) {
	ids := strings.Split(req.URL.Path[len("/v1/info/"):], "/")
	if len(ids) < 1 {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	id, err := strconv.ParseInt(ids[0], 10, 0)
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	dbEntry, ok := data.entries[int(id)]
	if !ok {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	var pkg dbPackage
	if len(ids) >= 2 {
		var ok bool
		pkg, ok = dbEntry.versions[ids[1]]
		if !ok {
			rw.Write([]byte("{\"error\": \"not found\"}"))
			return
		}
	} else {
		pkg = dbEntry.newest()
	}

	json.NewEncoder(rw).Encode(pkg.pkg)
}

func latest(rw http.ResponseWriter, req *http.Request) {
	id, err := strconv.ParseInt(req.URL.Path[len("/v1/latest/"):], 10, 0)
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	dbEntry, ok := data.entries[int(id)]
	if !ok {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	pkg := dbEntry.newest()

	file, err := os.Open("data/" + pkg.manifest.Hash + ".zip")
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}
	defer file.Close()

	rw.Header().Set("content-disposition", "attachment; filename=\""+strings.ReplaceAll(strings.ReplaceAll(pkg.pkg.Name, "\\", "\\\\"), "\"", "\\\"")+".zip\"")
	io.Copy(rw, file)
}

func versions(rw http.ResponseWriter, req *http.Request) {
	id, err := strconv.ParseInt(req.URL.Path[len("/v1/versions/"):], 10, 0)
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	dbEntry, ok := data.entries[int(id)]
	if !ok {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	result := make([]string, 0, len(dbEntry.versions))
	for ver := range dbEntry.versions {
		result = append(result, ver)
	}

	json.NewEncoder(rw).Encode(result)
}

func get(rw http.ResponseWriter, req *http.Request) {
	ids := strings.Split(req.URL.Path[len("/v1/download/"):], "/")
	if len(ids) < 2 {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	id, err := strconv.ParseInt(ids[0], 10, 0)
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	version := ids[1]

	dbEntry, ok := data.entries[int(id)]
	if !ok {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	pkg, ok := dbEntry.versions[version]
	if !ok {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}

	file, err := os.Open("data/" + pkg.manifest.Hash + ".zip")
	if err != nil {
		rw.Write([]byte("{\"error\": \"not found\"}"))
		return
	}
	defer file.Close()

	rw.Header().Set("content-disposition", "attachment; filename=\""+strings.ReplaceAll(strings.ReplaceAll(pkg.pkg.Name, "\\", "\\\\"), "\"", "\\\"")+".zip\"")
	io.Copy(rw, file)
}

func register(rw http.ResponseWriter, req *http.Request) {
	var body struct {
		ID   int    `json:"id"`
		URL  string `json:"url"`
		Path string `json:"path"`
	}

	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		rw.Write([]byte("{\"error\": \"invalid data\"}"))
		return
	}

	if body.ID == 0 {
		err = data.newEntryAndID(body.URL, body.Path)
	} else {
		err = data.newEntry(body.ID, body.URL, body.Path)
	}
	if err != nil {
		result := struct {
			Error  string `json:"error"`
			Detail string `json:"detail"`
		}{
			Error:  "invalid data",
			Detail: err.Error(),
		}

		json.NewEncoder(rw).Encode(result)
		return
	}

	rw.Write([]byte("{\"success\": true}"))
}
