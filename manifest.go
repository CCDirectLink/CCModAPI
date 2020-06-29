package main

import (
	"encoding/json"
	"io/ioutil"
)

type ManifestEntry struct {
	ModID    int
	URL      string
	Path     string
	Hash     string
	Approved bool
}

type Manifest []ManifestEntry

const manifestFile = "data/manifest.json"

func LoadManifest() (Manifest, error) {
	raw, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return Manifest{}, err
	}

	var result Manifest
	err = json.Unmarshal(raw, &result)
	return result, err
}

func (m Manifest) Save() error {
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(manifestFile, raw, 0666)
}

func (m *Manifest) Add(manifest ...ManifestEntry) error {
	*m = append(*m, manifest...)
	return m.Save()
}
