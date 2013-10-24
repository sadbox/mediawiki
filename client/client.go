package main

import (
    "github.com/sadbox/go-mwapi"
)

func main() {
    client, err := mwapi.New("http://en.wikipedia.org/w/api.php")
    if err != nil {
        panic(err)
    }

    client.Username = "USERNAME"
    client.Password = "PASSWORD"
    //client.Domain = "example.org"

    err = client.Login()
    if err != nil {
        panic(err)
    }

    client.Logout()
}
