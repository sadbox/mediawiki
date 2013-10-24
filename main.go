package mwapi

import (
    "net/url"
    "net/http"
    "net/http/cookiejar"
    "encoding/xml"
    "io/ioutil"
    "errors"
)

type MWApi struct {
    Username string
    Password string
    Domain string
    url *url.URL
    client *http.Client
    format string
    edittoken string
}

type Values map[string]string

type OuterLogin struct {
    Login LoginResponse `xml:"login"`
}

type LoginResponse struct {
    Result string `xml:"result,attr"`
    Token string `xml:"token,attr"`
}

type Query struct {
    Pages []Page `xml:"query>pages>page"`
}

type OuterEdit struct {
    EditResponse Edit `xml:"edit"`
}

type Edit struct {
    Result string `xml:"result,attr"`
    PageId string `xml:"pageid,atrr"`
    Title string `xml:"title,attr"`
    OldRevId int `xml:"oldrevid,attr"`
    NewRevId int `xml:"newrevid,attr"`
}

type Page struct {
    PageId string `xml:"pageid,attr"`
    Ns string `xml:"ns,attr"`
    Title string `xml:"title,attr"`
    Touched string `xml:"touched,attr"`
    Lastrevid int64 `xml:"lastrevid,attr"`
    Counter int `xml:"counter,attr"`
    Length int `xml:"length,attr"`
    Edittoken string `xml:"edittoken,attr"`
}

func New(wikiUrl string) (*MWApi, error) {
    cookiejar, err := cookiejar.New(nil)
    if err != nil {
        return nil, err
    }

    client := http.Client{
        Transport: nil,
        CheckRedirect: nil,
        Jar: cookiejar,
    }

    clientUrl, err := url.Parse(wikiUrl)
    if err != nil {
        return nil, err
    }

    return &MWApi{
        url: clientUrl,
        client: &client,
        format: "xml",
    }, nil
}

func (m *MWApi) PostForm(query url.Values) ([]byte, error) {
    resp, err := m.client.PostForm(m.url.String(), query)
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

func (m *MWApi) Login() (error) {
    if m.Username == "" || m.Password == "" {
        return errors.New("Username or password not set.")
    }

    query := m.url.Query()
    query.Set("action", "login")
    query.Set("lgname", m.Username)
    query.Set("lgpassword", m.Password)
    query.Set("format", m.format)

    if m.Domain != "" {
        query.Set("lgdomain", m.Domain)
    }

    body, err := m.PostForm(query)
    if err != nil {
        return err
    }

    var response OuterLogin
    err = xml.Unmarshal(body, &response)
    if err != nil {
        return err
    }

    if response.Login.Result == "Success" {
        return nil
    } else if response.Login.Result != "NeedToken" {
        return errors.New("Error logging in: "+response.Login.Result)
    }

    // Need to use the login token
    query.Set("lgtoken", response.Login.Token)

    body, err = m.PostForm(query)
    if err != nil {
        return err
    }

    err = xml.Unmarshal(body, &response)
    if err != nil {
        return err
    }

    if response.Login.Result == "Success" {
        return nil
    } else {
        return errors.New("Error logging in: "+response.Login.Result)
    }
}

func (m *MWApi) GetEditToken() error {
    query := m.url.Query()
    query.Set("action", "query")
    query.Set("prop", "info|revisions")
    query.Set("intoken", "edit")
    query.Set("titles", "Main Page")
    query.Set("format", "xml")
    body, err := m.PostForm(query)
    var response Query
    err = xml.Unmarshal(body, &response)
    if err != nil {
        return err
    }
    m.edittoken = response.Pages[0].Edittoken
    return nil
}

func (m *MWApi) Logout() {
    query := m.url.Query()
    query.Set("action", "logout")
    m.PostForm(query)
}

func (m *MWApi) Edit(values Values) error {
    query := m.url.Query()
    query.Set("action", "edit")
    query.Set("format", "xml")
    for key, value := range values {
        query.Set(key, value)
    }
    if m.edittoken == "" {
        err := m.GetEditToken()
        if err != nil {
            return err
        }
    }
    query.Set("token", m.edittoken)
    body, err := m.PostForm(query)
    if err != nil {
        return err
    }
    var response OuterEdit
    err = xml.Unmarshal(body, &response)
    if err != nil {
        return err
    }
    if response.EditResponse.Result == "Success" {
        return nil
    } else {
        return errors.New(response.EditResponse.Result)
    }
}
