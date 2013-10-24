package main

import (
	"encoding/xml"
	"github.com/sadbox/go-mwapi"
	"io/ioutil"
)

type Config struct {
	WikiURL  string
	Username string
	Password string
	Domain   string
}

func main() {
	var config Config
	xmlFile, err := ioutil.ReadFile("config.xml")
	if err != nil {
		panic(err)
	}
	xml.Unmarshal(xmlFile, &config)

	client, err := mwapi.New(config.WikiURL)
	if err != nil {
		panic(err)
	}

	client.Username = config.Username
	client.Password = config.Password
	client.Domain = config.Domain

	err = client.Login()
	if err != nil {
		panic(err)
	}

	editConfig := mwapi.Values{
		"title":   "NOC Test",
		"summary": "TESTING",
		"text":    "CLEARD FOR TESTING",
	}
	err = client.Edit(editConfig)
	if err != nil {
		panic(err)
	}
	client.Logout()
}
