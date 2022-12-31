package app

import (
	"encoding/json"
	"fmt"
	"go-tiny-url/common"
	"go-tiny-url/storage"
	"log"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// App encapsulates Env, Router and middleware
type App struct {
	Router *mux.Router
	validate *validator.Validate
	Middlewares *Middleware
	config *storage.Env
}

type shortenReq struct {
	URL string `json:"url" validate:"required"`
	ExpirationInMinutes int64 `json:"expiration_in_minutes" validate:"min=0"`
}

type shortLinkResp struct {
	ShortLink string `json:"short_link"`
}

// Initialize is initilization of App
func (a * App) Initialize(env *storage.Env) {
	// set log formmater
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	a.Router = mux.NewRouter()
	a.validate = validator.New()
	a.Middlewares = &Middleware{}
	a.config = env

	a.initRoutes()
}

func (a * App) initRoutes() {
	m := alice.New(a.Middlewares.LoggingHandler, a.Middlewares.RecoverHandler)

	a.Router.Handle("/api/shorten", 
			m.ThenFunc(a.createShortLink)).Methods("POST")
	a.Router.Handle("/api/info", 
			m.ThenFunc(a.getShortLinkInfo)).Methods("GET")
	a.Router.Handle("/{shortLink:[a-zA-Z0-9]{1,11}}", 
			m.ThenFunc(a.redirect)).Methods("GET")
}

func (a * App) createShortLink(w http.ResponseWriter, r *http.Request) {
	var req shortenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, common.StatusError{http.StatusBadRequest, 
						fmt.Errorf("parse parameters failed %v", r.Body)})
		return
	}
	if err := a.validate.Struct(req); err != nil {
		respondWithError(w, common.StatusError{http.StatusBadRequest,
							fmt.Errorf("validate parameters failed %v", req)})
		return 
	}
	defer r.Body.Close()
	fmt.Printf("%+v\n", req)

	s, err := a.config.St.Shorten(req.URL, req.ExpirationInMinutes)
	if err != nil {
		respondWithError(w, err)
	} else {
		respondWithJson(w, http.StatusCreated, shortLinkResp{ShortLink: s})
	}
}

func (a * App) getShortLinkInfo(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	s := vals.Get("shortLink")
	
	d, err := a.config.St.ShortLinkInfo(s)
	if err != nil {
		respondWithError(w, err)
	} else {
		respondWithJson(w, http.StatusOK, d)
	}
}

func (a * App) redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	
	url, err := a.config.St.Unshorten(vars["shortLink"])
	if err != nil {
		respondWithError(w, err)
	} else {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// Run starts listener and server
func (a * App) Run (addr string) {
	log.Fatal(http.ListenAndServe(addr, a.Router))
}


func respondWithError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case common.Error:
		log.Printf("HTTP %d - %s", e.Status(), e)
		respondWithJson(w, e.Status(), e.Error())
	default:
		respondWithJson(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	resp, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(resp)
}
