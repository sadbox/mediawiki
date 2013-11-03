package mediawiki

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const (
	editTokenReponse = `{"query":{"pages":{"15580374":{"pageid":15580374,"ns":0,"title":"Main Page","contentmodel":"wikitext","pagelanguage":"en","touched":"2013-11-02T14:08:05Z","lastrevid":574690625,"counter":"","length":6391,"starttimestamp":"2013-11-02T15:24:43Z","edittoken":"+\\","revisions":[{"revid":574690625,"parentid":574690493,"minor":"","user":"Tariqabjotu","timestamp":"2013-09-27T03:10:17Z","comment":"removing unnecessary pipe"}]}}}}`
	firstLogin       = `{"login":{"result":"NeedToken","token":"8f48670ddc7fa9d5fa7e7fa2ae147e80","cookieprefix":"wikidb","sessionid":"927e0d298f6f3b5bb21228803fd9c0eb"}}`
	secondLogin      = `{"login":{"result":"Success","token":"8f48670ddc7fa9d5fa7e7fa2ae147e80","cookieprefix":"wikidb","sessionid":"927e0d298f6f3b5bb21228803fd9c0eb"}}`
	failedLogin      = `{"login":{"result":"ERROR THING","token":"8f48670ddc7fa9d5fa7e7fa2ae147e80","cookieprefix":"wikidb","sessionid":"927e0d298f6f3b5bb21228803fd9c0eb"}}`
    readPage = `{"query-continue":{"revisions":{"rvcontinue":574690493}},"query":{"pages":{"15580374":{"pageid":15580374,"ns":0,"title":"Main Page","revisions":[{"user":"Tariqabjotu","timestamp":"2013-09-27T03:10:17Z","comment":"removing unnecessary pipe","contentformat":"text/x-wiki","contentmodel":"wikitext","*":"FULL PAGE TEXT"}]}}}}`
)

type Test struct {
	ts     *httptest.Server
	client *MWApi
}

func (t *Test) TearDown() {
	t.ts.Close()
}

func BuildUp(response string, t *testing.T) *Test {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, response)
	}))
	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("Error in BuildUp: %s", err)
	}
	return &Test{ts: ts, client: client}
}

func TestGetEditToken(t *testing.T) {
	test := BuildUp(editTokenReponse, t)
	defer test.TearDown()

	err := test.client.GetEditToken()
	if err != nil {
		t.Errorf("Could not get edit token: %s", err.Error())
	} else {
		t.Log("Got edit token")
	}
	if test.client.edittoken == `+\` {
		t.Log("Edit token correct")
	} else {
		t.Errorf("Incorrect edit token: %s", test.client.edittoken)
	}
}

func TestLogin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		if r.Form.Get("lgtoken") == "" {
			fmt.Fprintln(w, firstLogin)
		} else {
			fmt.Fprintln(w, secondLogin)
		}
	}))
	defer ts.Close()
	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("Error creating client: %s", err.Error())
	}
	client.Password = "asdf"
	client.Username = "asdf"
	err = client.Login()
	if err != nil {
		t.Error("Client failed to login: %s", err.Error())
	} else {
		t.Log("Client logged in successfully.")
	}
}

func TestLoginNoPW(t *testing.T) {
	test := BuildUp(failedLogin, t)
	defer test.TearDown()
	err := test.client.Login()
	if err != nil {
		t.Log("Client refused to login with password.")
	} else {
		t.Error("Client did not a return an error with no password set")
	}
}

func TestLoginFailed(t *testing.T) {
	test := BuildUp(failedLogin, t)
	defer test.TearDown()
	test.client.Username = "asdf"
	test.client.Password = "JKLa"
	err := test.client.Login()
	if err != nil {
		t.Logf("Failed to log in: %s", err.Error())
	} else {
		t.Error("Client logged in successfully to incorrect response")
	}
}

func TestPostForm(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		if r.Header.Get("user-agent") != "go-mediawiki https://github.com/sadbox/go-mediawiki" {
			fmt.Fprintln(w, "USERAGENT")
		} else {
			fmt.Fprintln(w, r.Form.Get("KEY"))
		}
	}))
	defer ts.Close()
	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("Error creating client: %s", err.Error())
	}
	value, err := client.postForm(url.Values{"KEY": []string{"VALUE"}})
	if err != nil {
		t.Errorf("Error posting data: %s", err.Error())
	}
	returnValue := strings.TrimSpace(string(value))
	if returnValue == "VALUE" {
		t.Log("Successfully posted to web service.")
	} else if returnValue == "USERAGENT" {
		t.Error("Wrong useragent used!")
	} else {
		t.Error("Did not retrieve right value from web service")
	}
}

func TestAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		if r.Form.Get("KEY") != "VALUE" || r.Form.Get("OTHER KEY") != "OTHER VALUE" || r.Form.Get("format") != "json" {
			fmt.Fprintln(w, `{"status":"FAIL"}`)
		} else {
			fmt.Fprintln(w, `{"status":"PASS"}`)
		}
	}))
	defer ts.Close()
	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("Error creating client: %s", err.Error())
	}
	body, _, err := client.API(Values{"KEY": "VALUE", "OTHER KEY": "OTHER VALUE"})
	if err != nil {
		t.Fatalf("API() returned a non-nil error: %s", err.Error())
	}
	if strings.TrimSpace(string(body)) == `{"status":"PASS"}` {
		t.Log("Successfully posted all information via API() call")
	} else {
		t.Error("Did not get PASS back from test server when posting via API()")
	}
}

func TestRead(t *testing.T) {
    test := BuildUp(readPage, t)
    defer test.TearDown()
    page, err := test.client.Read("TESTING PAGE")
    if err != nil {
        t.Fatal("Unable to read page: %s", err)
    }
    if page.Body != "FULL PAGE TEXT" {
        t.Error("Page content not correct")
    }
}
