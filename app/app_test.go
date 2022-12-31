package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-tiny-url/app"
	"go-tiny-url/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
)

const (
	expTime = 60
	longUrl = "https://www.baidu.com"
	shortLink = "IFHzaO"
	shortLinkInfo = `{
		"url": "https://www.baidu.com", 
		"created_at": "2022-12-30 21:50:01.134758059 -0500 EST m=+12.075415907",
		"expiration_in_minutes": 60
	}`
)

type storageMock struct {
	mock.Mock
}

var a app.App
var mockR * storageMock

func (s *storageMock) Shorten(url string, exp int64) (string, error) {
	args := s.Called(url, exp)
	return args.String(0), args.Error(1)
}


func (s *storageMock) Unshorten(eid string) (string, error) {
	args := s.Called(eid)
	return args.String(0), args.Error(1)
}

func (s *storageMock) ShortLinkInfo(eid string) (interface{}, error) {
	args := s.Called(eid)
	return args.String(0), args.Error(1)
}

func init() {
	a = app.App{}
	mockR = new(storageMock)
	a.Initialize(&storage.Env{St: mockR})
}

func TestCreateShortLink(t *testing.T) {
	var jsonStr = []byte(`{
		"url": "https://www.baidu.com",
		"expiration_in_minutes": 60
	}`)
	req, err := http.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal("Should be able to create request. ", err)
	}
	req.Header.Set("Content-Type", "application/json")

	mockR.On("Shorten", longUrl, int64(expTime)).Return(shortLink, nil).Once()
	rw := httptest.NewRecorder()
	a.Router.ServeHTTP(rw, req)

	if rw.Code != http.StatusCreated {
		t.Fatalf("Excepted receive %d. Got %d", http.StatusCreated, rw.Code)
	}

	resp := struct {
		ShortLink string `json:"short_link"`
	}{}

	if err := json.NewDecoder(rw.Body).Decode(&resp); err != nil {
		t.Fatal("Should decode the response")
	}

	if resp.ShortLink != shortLink {
		t.Fatalf("Expect to receive %s. Got %s", shortLink, resp.ShortLink)
	}
}

func TestRedirect(t *testing.T) {
	r := fmt.Sprintf("/%s", shortLink)
	req, err := http.NewRequest("GET", r, nil) 
	if err != nil {
		t.Fatal("Should be able to create a request.", err)
	}

	mockR.On("Unshorten", shortLink).Return(longUrl, nil).Once()
	rw := httptest.NewRecorder()
	a.Router.ServeHTTP(rw, req)

	if rw.Code != http.StatusTemporaryRedirect {
		t.Fatalf("Expect to receive %d. Got %d", http.StatusTemporaryRedirect, rw.Code)
	}
}