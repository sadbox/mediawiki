// Copyright 2013 James McGuire
// This code is covered under the MIT License
// Please refer to the LICENSE file in the root of this
// repository for any information.

// Mediawiki API
//
// Please see http://www.mediawiki.org/wiki/API:Main_page
// for any API specific information or refer to any of the
// functions defined for the MWApi struct for information
// regarding this specific implementation.
//
// The client subdirectory contains an example application
// that uses this API.
package mediawiki

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// The main mediawiki API struct, this is generated via mwapi.New()
type MWApi struct {
	Username  string
	Password  string
	Domain    string
	url       *url.URL
	client    *http.Client
	format    string
	edittoken string
}

// This is used for passing data to the mediawiki API via key=value in a POST
type Values map[string]string

// Unmarshal login data...
type outerLogin struct {
	Login loginResponse
}

type loginResponse struct {
	Result string
	Token  string
}

// Unmarshall response from page edits...
type outerEdit struct {
	Edit edit
}

type edit struct {
	Result   string
	PageId   int
	Title    string
	OldRevId int
	NewRevId int
}

// General query response from mediawiki
type Query struct {
	Query query
}

type query struct {
	Pages map[string]page
}

type page struct {
	Pageid    int
	Ns        float64
	Title     string
	Touched   string
	Lastrevid float64
	Counter   float64
	Length    float64
	Edittoken string
}

// Generate a new mediawiki API struct
//
// Example: mwapi.New("http://en.wikipedia.org/w/api.php")
// Returns errors if the URL is invalid
func New(wikiUrl string) (*MWApi, error) {
	cookiejar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           cookiejar,
	}

	clientUrl, err := url.Parse(wikiUrl)
	if err != nil {
		return nil, err
	}

	return &MWApi{
		url:    clientUrl,
		client: &client,
		format: "json",
	}, nil
}

// This will automatically add the user agent and encode the http request properly
func (m *MWApi) postForm(query url.Values) ([]byte, error) {
	request, err := http.NewRequest("POST", m.url.String(), strings.NewReader(query.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("user-agent", "go-mediawiki https://github.com/sadbox/go-mediawiki")
	resp, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Login to the Mediawiki Website
//
// This will throw an error if you didn't define a username
// or password.
func (m *MWApi) Login() error {
	if m.Username == "" || m.Password == "" {
		return errors.New("Username or password not set.")
	}

	query := Values{
		"action":     "login",
		"lgname":     m.Username,
		"lgpassword": m.Password,
	}

	if m.Domain != "" {
		query["lgdomain"] = m.Domain
	}

	body, _, err := m.API(query)
	if err != nil {
		return err
	}

	var response outerLogin
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Login.Result == "Success" {
		return nil
	} else if response.Login.Result != "NeedToken" {
		return errors.New("Error logging in: " + response.Login.Result)
	}

	// Need to use the login token
	query["lgtoken"] = response.Login.Token

	body, _, err = m.API(query)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	if response.Login.Result == "Success" {
		return nil
	} else {
		return errors.New("Error logging in: " + response.Login.Result)
	}
}

// Get an edit token
//
// This is necessary for editing any page.
//
// The Edit() function will call this automatically
// but it is available if you want to make direct
// calls to API().
func (m *MWApi) GetEditToken() error {
	query := Values{
		"action":  "query",
		"prop":    "info|revisions",
		"intoken": "edit",
		"titles":  "Main Page",
	}

	body, _, err := m.API(query)
	if err != nil {
		return err
	}
	var response Query
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	for _, value := range response.Query.Pages {
		m.edittoken = value.Edittoken
		break
	}
	return nil
}

// Log out of the mediawiki website
//
// Doesn't really matter, but good form.
func (m *MWApi) Logout() {
	m.API(Values{"action": "logout"})
}

// Edit a page
//
// This function will automatically grab an Edit Token if there
// is not one currently stored.
//
// Example:
//
//  editConfig := mediawiki.Values{
//      "title":   "SOME PAGE",
//      "summary": "THIS IS WHAT SHOWS UP IN THE LOG",
//      "text":    "THE ENTIRE TEXT OF THE PAGE",
//  }
//  err = client.Edit(editConfig)
func (m *MWApi) Edit(values Values) error {
	if m.edittoken == "" {
		err := m.GetEditToken()
		if err != nil {
			return err
		}
	}
	query := Values{
		"action": "edit",
		"token":  m.edittoken,
	}
	body, _, err := m.API(query, values)
	if err != nil {
		return err
	}

	var response outerEdit
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Edit.Result == "Success" {
		return nil
	} else {
		return errors.New(response.Edit.Result)
	}
}

// Request a wiki page and it's metadata.
func (m *MWApi) Read(pageName string) {
	return
}

// A generic interface to the Mediawiki API
// Refer to the mediawiki API reference for any information regarding
// what to pass to this function
//
// This is used by all internal functions to interact with the API
//
// The second return is simply the json data decoded in to an empty interface
// that can be used by something like https://github.com/jmoiron/jsonq
func (m *MWApi) API(values ...Values) ([]byte, interface{}, error) {
	query := m.url.Query()
	for _, valuemap := range values {
		for key, value := range valuemap {
			query.Set(key, value)
		}
	}
	query.Set("format", m.format)
	body, err := m.postForm(query)
	if err != nil {
		return nil, nil, err
	}
	var unmarshalto interface{}
	err = json.Unmarshal(body, &unmarshalto)
	if err != nil {
		return nil, nil, err
	}
	return body, unmarshalto, nil
}
