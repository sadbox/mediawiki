package main

import (
	"encoding/xml"
	"github.com/sadbox/go-mwapi"
	"io/ioutil"
)

var config Config

type Config struct {
	WikiURL  string
	Username string
	Password string
	Domain   string
}

func init() {
	// None of this is relevant to the actual wiki API, just getting configurations
	xmlFile, err := ioutil.ReadFile("config.xml")
	if err != nil {
		panic(err)
	}
	xml.Unmarshal(xmlFile, &config)
}

func main() {
	client, err := mwapi.New(config.WikiURL)
	if err != nil {
		panic(err)
	}

    // The username and passsword are required
	client.Username = config.Username
	client.Password = config.Password
    // But the domain is not
	client.Domain = config.Domain

	err = client.Login()
	if err != nil {
		panic(err)
	}
    // This is probably not required
	defer client.Logout()

	editConfig := mwapi.Values{
		"title":   "SOME PAGE",
		"summary": "THIS IS WHAT SHOWS UP IN THE LOG",
		"text":    "THE ENTIRE TEXT OF THE PAGE",
	}
	err = client.Edit(editConfig)
	if err != nil {
		panic(err)
	}
}
