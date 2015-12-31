// Copyright 2013 James McGuire
// This code is covered under the MIT License
// Please refer to the LICENSE file in the root of this
// repository for any information.

// Package mediawiki provides a wrapper for interacting with the Mediawiki API
//
// Please see http://www.mediawiki.org/wiki/API:Main_page
// for any API specific information or refer to any of the
// functions defined for the MWApi struct for information
// regarding this specific implementation.
//
// The examples/ subdirectory contains an example application
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

// MWApi is used to interact with the MediaWiki server.
type MWApi struct {
	username      string
	password      string
	Domain        string
	userAgent     string
	url           *url.URL
	client        *http.Client
	format        string
	edittoken     string
	UseBasicAuth  bool
	BasicAuthUser string
	BasicAuthPass string
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

// Response is a struct used for unmarshaling the MediaWiki JSON response.
type Response struct {
	Query struct {
		// The json response for this part of the struct is dumb.
		// It will return something like { '23': { 'pageid': 23 ...
		//
		// As a workaround you can use GenPageList which will create
		// a list of pages from the map.
		Pages    map[string]Page
		PageList []Page
	}
}

// GenPageList generates PageList from Pages to work around the sillyness in
// the MediaWiki API.
func (r *Response) GenPageList() {
	r.Query.PageList = []Page{}
	for _, page := range r.Query.Pages {
		r.Query.PageList = append(r.Query.PageList, page)
	}
}

// A Page represents a MediaWiki page and its metadata.
type Page struct {
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
		// Take note, MediaWiki literally returns { '*':
		Body      string `json:"*"`
		User      string
		Timestamp string
		Comment   string
	}
	Imageinfo []struct {
		Url            string
		Descriptionurl string
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

// Helper function for translating MediaWiki errors in to Golang errors.
func checkError(response []byte) error {
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

// New generates a new MediaWiki API (MWApi) struct.
//
// Example: mediawiki.New("http://en.wikipedia.org/w/api.php", "My Mediawiki Bot")
// Returns errors if the URL is invalid
func New(wikiURL, userAgent string) (*MWApi, error) {
	cookiejar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           cookiejar,
	}

	clientURL, err := url.Parse(wikiURL)
	if err != nil {
		return nil, err
	}

	return &MWApi{
		url:       clientURL,
		client:    &client,
		format:    "json",
		userAgent: "mediawiki (Golang) https://github.com/sadbox/mediawiki " + userAgent,
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
	if m.UseBasicAuth {
		request.SetBasicAuth(m.BasicAuthUser, m.BasicAuthPass)
	}
	resp, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err = checkError(body); err != nil {
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

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	response.GenPageList()

	if len(response.Query.PageList) < 1 {
		return nil, errors.New("no file found")
	}
	page := response.Query.PageList[0]
	if len(page.Imageinfo) < 1 {
		return nil, errors.New("no file found")
	}
	fileurl := page.Imageinfo[0].Url

	// Then return the body of the response
	request, err := http.NewRequest("GET", fileurl, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("user-agent", m.userAgent)
	if m.UseBasicAuth {
		request.SetBasicAuth(m.BasicAuthUser, m.BasicAuthPass)
	}

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
	if m.UseBasicAuth {
		request.SetBasicAuth(m.BasicAuthUser, m.BasicAuthPass)
	}

	resp, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = checkError(body); err != nil {
		return err
	}

	var response uploadResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if !(response.Upload.Result == "Success" || response.Upload.Result == "Warning") {
		return errors.New(response.Upload.Result)
	}
	return nil
}

// Login to the Mediawiki Website.
func (m *MWApi) Login(username, password string) error {
	if username == "" {
		return errors.New("empty username supplied")
	}
	if password == "" {
		return errors.New("empty password supplied")
	}
	m.username = username
	m.password = password

	query := map[string]string{
		"action":     "login",
		"lgname":     m.username,
		"lgpassword": m.password,
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

	if response.Login.Result != "Success" {
		return errors.New("Error logging in: " + response.Login.Result)
	}
	return nil
}

// GetEditToken retrieves an edit token from the MediaWiki site and saves it.
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
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	response.GenPageList()
	if len(response.Query.PageList) < 1 {
		return errors.New("no pages returned for edittoken query")
	}
	m.edittoken = response.Query.PageList[0].Edittoken
	return nil
}

// Logout of the MediaWiki website
func (m *MWApi) Logout() {
	m.API(map[string]string{"action": "logout"})
}

// Edit a page.
//
// This function will request an edit token if the MWApi struct doesn't already
// contain one.
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

	if response.Edit.Result != "Success" {
		return errors.New(response.Edit.Result)
	}
	return nil
}

// Read returns a Response which contains the contents of a page.
// Only the most recent revision is fetched.
func (m *MWApi) Read(pageName string) (*Response, error) {
	query := map[string]string{
		"action":  "query",
		"prop":    "revisions",
		"titles":  pageName,
		"rvlimit": "1",
		"rvprop":  "content|timestamp|user|comment",
	}
	body, err := m.API(query)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// API is a generic interface to the Mediawiki API.
// Refer to the MediaWiki API reference for details.
//
// This is used by all internal functions to interact with the API.
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
