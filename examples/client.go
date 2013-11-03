package main

import (
	"encoding/xml"
	"github.com/sadbox/go-mediawiki"
	"io/ioutil"
    "fmt"
    "log"
    "io"
    "os"
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
		log.Fatal(err)
	}
	xml.Unmarshal(xmlFile, &config)
}

func main() {
    // CREATE A NEW API STRUCT
	client, err := mediawiki.New(config.WikiURL, "MY TEST APP")
	if err != nil {
		log.Fatal(err)
	}


    // LOGIN
	// The username and passsword are required
	client.Username = config.Username
	client.Password = config.Password
	// But the domain is not
	client.Domain = config.Domain

	err = client.Login()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Logout()


    // READ A PAGE
    page, err := client.Read("Main Page")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(page.Body)


    // DOWNLOAD A FILE
    src, err := client.Download("File:SomeFile.png")
    if err != nil {
        log.Fatal(err)
    }
    defer src.Close()

    dst, err := os.Create("/tmp/test_download")
    if err != nil {
        log.Fatal(err)
    }

    _, err = io.Copy(dst, src)
    if err != nil {
        log.Fatal(err)
    }

    fi, err := os.Stat("/tmp/test_download")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(fi.Name(), fi.Size())


    // EDIT A PAGE
	editConfig := mediawiki.Values{
		"title":   "SOME PAGE",
		"summary": "THIS IS WHAT SHOWS UP IN THE LOG",
		"text":    "THE ENTIRE TEXT OF THE PAGE",
	}

	err = client.Edit(editConfig)

	if err != nil {
		log.Fatal(err)
	}
}
