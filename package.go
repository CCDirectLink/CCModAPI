package main

//Package data.
type Package struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	DisplayName  string            `json:"ccmodHumanName"`
	Dependencies map[string]string `json:"ccmodDependencies"`
}
