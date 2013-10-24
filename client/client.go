package main

import (
    "github.com/sadbox/go-mwapi"
    "encoding/xml"
    "io/ioutil"
)

type Config struct {
    WikiURL string
    Username string
    Password       string
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
    //client.Domain = "example.org"

    err = client.Login()
    if err != nil {
        panic(err)
    }
    client.GetEditToken()

    client.Logout()
}
