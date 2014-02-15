package mediawiki

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// MWApi is used to interact with the mediawiki server.
type MWApi struct {
	Username      string
	Password      string
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
		PageID   int
		Title    string
		OldRevID int
		NewRevID int
	}
}

// Response is a struct used for unmarshaling the mediawiki JSON
// response to.
//
// It should be particularly useful when API needs to be called
// directly.
type Response struct {
	Query Query
}

// Query is used just for Responses, but it needs to be a separate struct
// so that an UnmarshalJSON function can be added to it.
type Query struct {
	Pages []Page
}

// UnmarshalJSON is used to unmarshal the mediawiki pages in to a list
func (q *Query) UnmarshalJSON(b []byte) error {
	tempData := struct{ Pages map[string]Page }{make(map[string]Page)}
	err := json.Unmarshal(b, &tempData)
	if err != nil {
		return err
	}
	q.Pages = []Page{}
	for _, page := range tempData.Pages {
		q.Pages = append(q.Pages, page)
	}
	return nil
}

// Page is a mediawiki page and it's related metadata
type Page struct {
	PageID    int
	Ns        int
	Title     string
	Touched   string
	LastRevID int
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
		Comment   string
	}
	ImageInfo []struct {
		URL            string
		DescriptionURL string
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
