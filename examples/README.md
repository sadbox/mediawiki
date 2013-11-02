Basic Usage:
============

Creating a New Connection
----------
All you need to create a new MWApi struct is the URL of the site that you want to connect to
```Go
client, err := mediawiki.New(`http://en.wikipedia.org/w/api.php`)
if err != nil {
    // Handle the error
}
```

Logging In
-----------
You do not always have to log in, but there is a helper for storing the cookies and such necessary.
```Go
// The username and passsword are required
client.Username = "go-bot"
client.Password = "go-bots password"
// But the domain is not
client.Domain = "go-bot.org"

err = client.Login()
if err != nil {
    // Handle the error
}
defer client.Logout()
```

Edit a Page
-----------
Editing a page will require using the mediawiki.Values type. This is necessary because there are a large number of possibilities for what you might want to do during an edit.
```Go
editConfig := mediawiki.Values{
        "title":   "SOME PAGE",
        "summary": "THIS IS WHAT SHOWS UP IN THE LOG",
        "text":    "THE ENTIRE TEXT OF THE PAGE",
}
err = client.Edit(editConfig)
if err != nil {
        // Handle the error
}
```

Generic API
-----------
In the same way you had to craft part of the request for editing, this will allow you to pass arbitrary post-values to mediawiki while still retaining the proper user-agent and login credentials.
```Go
body, unmarshaledJson, err = client.API(mediawiki.Values{"action": "logout"})
if err != nil {
    // Handle the error
}
```
