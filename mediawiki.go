// Copyright 2013 James McGuire
// This code is covered under the MIT License
// Please refer to the LICENSE file in the root of this
// repository for any information.

// go-mediawiki provides a wrapper for interacting with the Mediawiki API
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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
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
	userAgent string
	url       *url.URL
	client    *http.Client
	format    string
	edittoken string
}

// Unmarshal login data...
type outerLogin struct {
	Login struct {
		Result string
		Token  string
	}
}

// Unmarshall response from page edits...
type outerEdit struct {
	Edit struct {
		Result   string
		PageId   int
		Title    string
		OldRevId int
		NewRevId int
	}
}

// General query response from mediawiki
type mwQuery struct {
	Query struct {
		// The json response for this part of the struct is dumb.
		// It will return something like { '23': { 'pageid':....
		// So then the you to do this craziness with a map... but
		// basically means you're forced to extract your pages with
		// range instead of something sane. Sorry!
		Pages map[string]struct {
			Pageid    int
			Ns        int
			Title     string
			Touched   string
			Lastrevid int
			// Mediawiki will return '' for zero, this makes me sad.
			// If for some reason you need this value you'll have to
			// do some type assertion sillyness.
			Counter   interface{}
			Length    int
			Edittoken string
			Revisions []struct {
				// Take note, mediawiki literally returns { '*':
				Body      string `json:"*"`
				User      string
				Timestamp string
				comment   string
			}
			Imageinfo []struct {
				Url            string
				Descriptionurl string
			}
		}
	}
}

type mwError struct {
	Error struct {
		Code string
		Info string
	}
}

type uploadResponse struct {
	Upload struct {
		Result string
	}
}

// Helper function for translating mediawiki errors in to golang errors.
func CheckError(response []byte) error {
	var mwerror mwError
	err := json.Unmarshal(response, &mwerror)
	if err != nil {
		return nil
	} else if mwerror.Error.Code != "" {
		return errors.New(mwerror.Error.Code + ": " + mwerror.Error.Info)
	} else {
		return nil
	}
}

// Generate a new mediawiki API struct
//
// Example: mwapi.New("http://en.wikipedia.org/w/api.php", "My Mediawiki Bot")
// Returns errors if the URL is invalid
func New(wikiUrl, userAgent string) (*MWApi, error) {
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
		url:       clientUrl,
		client:    &client,
		format:    "json",
		userAgent: "go-mediawiki https://github.com/sadbox/go-mediawiki " + userAgent,
	}, nil
}

// This will automatically add the user agent and encode the http request properly
func (m *MWApi) postForm(query url.Values) ([]byte, error) {
	request, err := http.NewRequest("POST", m.url.String(), strings.NewReader(query.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("user-agent", m.userAgent)
	resp, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = CheckError(body); err != nil {
		return nil, err
	}

	return body, nil
}

// Download a file.
//
// Returns a readcloser that must be closed manually. Refer to the
// example app for additional usage.
func (m *MWApi) Download(filename string) (io.ReadCloser, error) {
	// First get the direct url of the file
	query := map[string]string{
		"action": "query",
		"prop":   "imageinfo",
		"iiprop": "url",
		"titles": filename,
	}

	body, err := m.API(query)
	if err != nil {
		return nil, err
	}

	var response mwQuery
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	var fileurl string
	for _, page := range response.Query.Pages {
		if len(page.Imageinfo) < 1 {
			return nil, errors.New("No file found")
		}
		fileurl = page.Imageinfo[0].Url
		break
	}

	// Then return the body of the response
	request, err := http.NewRequest("GET", fileurl, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("user-agent", m.userAgent)

	resp, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Upload a file
//
// This does a simple, but more error-prone upload. Mediawiki
// has a chunked upload version but it is only available in newer
// versions of the API.
//
// Automatically retrieves an edit token if necessary.
func (m *MWApi) Upload(dstFilename string, file io.Reader) error {
	if m.edittoken == "" {
		err := m.GetEditToken()
		if err != nil {
			return err
		}
	}

	query := map[string]string{
		"action":   "upload",
		"filename": dstFilename,
		"token":    m.edittoken,
		"format":   m.format,
	}

	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)

	for key, value := range query {
		err := writer.WriteField(key, value)
		if err != nil {
			return err
		}
	}

	part, err := writer.CreateFormFile("file", dstFilename)
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", m.url.String(), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("user-agent", m.userAgent)

	resp, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = CheckError(body); err != nil {
		return err
	}

	var response uploadResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Upload.Result == "Success" || response.Upload.Result == "Warning" {
		return nil
	} else {
		return errors.New(response.Upload.Result)
	}
}

// Login to the Mediawiki Website
//
// This will throw an error if you didn't define a username
// or password.
func (m *MWApi) Login() error {
	if m.Username == "" || m.Password == "" {
		return errors.New("Username or password not set.")
	}

	query := map[string]string{
		"action":     "login",
		"lgname":     m.Username,
		"lgpassword": m.Password,
	}

	if m.Domain != "" {
		query["lgdomain"] = m.Domain
	}

	body, err := m.API(query)
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

	body, err = m.API(query)
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
	query := map[string]string{
		"action":  "query",
		"prop":    "info|revisions",
		"intoken": "edit",
		"titles":  "Main Page",
	}

	body, err := m.API(query)
	if err != nil {
		return err
	}
	var response mwQuery
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
func (m *MWApi) Logout() {
	m.API(map[string]string{"action": "logout"})
}

// Edit a page
//
// This function will automatically grab an Edit Token if there
// is not one currently stored.
//
// Example:
//
//  editConfig := map[string]string{
//      "title":   "SOME PAGE",
//      "summary": "THIS IS WHAT SHOWS UP IN THE LOG",
//      "text":    "THE ENTIRE TEXT OF THE PAGE",
//  }
//  err = client.Edit(editConfig)
func (m *MWApi) Edit(values map[string]string) error {
	if m.edittoken == "" {
		err := m.GetEditToken()
		if err != nil {
			return err
		}
	}
	query := map[string]string{
		"action": "edit",
		"token":  m.edittoken,
	}
	body, err := m.API(query, values)
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
func (m *MWApi) Read(pageName string) (*mwQuery, error) {
	query := map[string]string{
		"action":  "query",
		"prop":    "revisions",
		"titles":  pageName,
		"rvlimit": "1",
		"rvprop":  "content|timestamp|user|comment",
	}
	body, err := m.API(query)

	var response mwQuery
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// A generic interface to the Mediawiki API
// Refer to the mediawiki API reference for any information regarding
// what to pass to this function.
//
// This is used by all internal functions to interact with the API
func (m *MWApi) API(values ...map[string]string) ([]byte, error) {
	query := m.url.Query()
	for _, valuemap := range values {
		for key, value := range valuemap {
			query.Set(key, value)
		}
	}
	query.Set("format", m.format)
	body, err := m.postForm(query)
	if err != nil {
		return nil, err
	}
	return body, nil
}
