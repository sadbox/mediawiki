package main

import (
    "sandbox/go-mwapi"
)

func main() {
    client, err := mwapi.New("http://en.wikipedia.org/w/api.php", "USERNAME", "PASSWORD")
    if err != nil {
        panic(err)
    }

    err = client.Login()
    if err != nil {
        panic(err)
    }

    client.Logout()
}
