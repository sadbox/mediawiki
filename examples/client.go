package main

import (
	"encoding/xml"
	"fmt"
	"github.com/sadbox/mediawiki"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
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
	// the username and passsword are required, but the domain is not
	client.Domain = config.Domain

	err = client.Login(config.Username, config.Password)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Logout()

	// READ A PAGE
	page, err := client.Read("Main Page")
	if err != nil {
		log.Fatal(err)
	}
	for _, rev := range page.Revisions {
		fmt.Println(rev.Body)
	}

	// UPLOAD A FILE
	file, err := os.Open("effective_go.pdf")
	if err != nil {
		log.Fatal(err)
	}
	err = client.Upload("SomeFile.pdf", file)
	if err != nil {
		log.Fatal(err)
	}

	// DOWNLOAD A FILE
	src, err := client.Download("File:SomeFile.pdf")
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
	userStrs := []string{"User:", config.Username, "/sandbox"}
	editConfig := map[string]string{
		"title": strings.Join(userStrs, ""),
		"summary": "THIS IS WHAT SHOWS UP IN THE LOG",
		"text": "THE ENTIRE TEXT OF THE PAGE",
	}

	err = client.Edit(editConfig)

	if err != nil {
		log.Fatal(err)
	}
}
