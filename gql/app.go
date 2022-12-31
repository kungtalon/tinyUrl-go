package gql

import (
	"encoding/json"
	"go-tiny-url/app"
	"go-tiny-url/common"
	"go-tiny-url/storage"
	"log"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"github.com/justinas/alice"
)


var ShortenInput = graphql.NewObject(graphql.ObjectConfig{
	Name: "ShortenInput",
	Fields: graphql.Fields{
		"url": &graphql.Field{Type: graphql.String},
		"expiration_in_minutes": &graphql.Field{Type: graphql.Int},
	},
})

var ShortLinkType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ShortLink",
	Fields: graphql.Fields{
		"short_link": &graphql.Field{Type: graphql.String},
	},
})

var ShortLinkDetailType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ShortLinkDetail",
	Fields: graphql.Fields{
		"url": &graphql.Field{Type: graphql.String},
		"created_at": &graphql.Field{Type: graphql.String},
		"expiration_in_minutes": &graphql.Field{Type: graphql.Int},
	},
})

type shortLinkResp struct {
	ShortLink string `json:"short_link"`
}

type urlDetailResp struct {
	URL string `json:"url"`
	CreatedAt string `json:"created_at"`
	ExpirationInMinutes int `json:"expiration_in_minutes"`
}

// App encapsulates Env, Router and middleware
type App struct {
	Router *mux.Router
	validate *validator.Validate
	Middlewares *app.Middleware
	schema graphql.Schema
	config *storage.Env
}


// Initialize is initilization of App
func (a * App) Initialize(env *storage.Env) {
	// set log formmater
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	a.Router = mux.NewRouter()
	a.validate = validator.New()
	a.Middlewares = &app.Middleware{}
	a.config = env
	a.createSchema()
	a.initRoutes()
}

func (a * App) initRoutes() {
	m := alice.New(a.Middlewares.LoggingHandler, a.Middlewares.RecoverHandler)

	h := handler.New(&handler.Config{
		Schema: &a.schema,
		Pretty: true,
		GraphiQL: true,
	})

	a.Router.Handle("/api", m.ThenFunc(h.ServeHTTP)).Methods("POST")
	a.Router.Handle("/{shortLink:[a-zA-Z0-9]{1,11}}", 
			m.ThenFunc(a.redirect)).Methods("GET")
}

func (a *App) createSchema() {
	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"shortLinkDetail": &graphql.Field{
				Type:       ShortLinkDetailType ,
				Description: "Get the short link detail",
				Args: graphql.FieldConfigArgument{
					"shortLink": &graphql.ArgumentConfig{Type: graphql.String},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					shortLink, ok := params.Args["shortLink"].(string)
					if ok {
						return a.getShortLinkInfo(shortLink)						
					}
					return nil, nil
				},
			},
		},
	 })

	mutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootMutation",
		Fields: graphql.Fields{
			"createShortLink": &graphql.Field{
				Type:  ShortLinkType ,
				Description: "Get the short link detail",
				Args: graphql.FieldConfigArgument{
					"url": &graphql.ArgumentConfig{Type: graphql.String},
					"expiration_in_minutes": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					url, urlOk := params.Args["url"].(string)
					expiration, expOk := params.Args["expiration_in_minutes"].(int)
					if urlOk && expOk {
						return a.createShortLink(url, int64(expiration))
					}
					return nil, nil
				},
			},
		},
	})

	a.schema, _ = graphql.NewSchema(
		graphql.SchemaConfig{
			Query:    query,
			Mutation: mutation,
		},
	)
}

func (a * App) createShortLink(url string, exp int64) (interface{}, error) {
	s, err := a.config.St.Shorten(url, exp)
	return shortLinkResp{ShortLink: s}, err
}

func (a* App) getShortLinkInfo(l string) (interface{}, error) {
	d, err := a.config.St.ShortLinkInfo(l)
	r := urlDetailResp{}
	json.Unmarshal([]byte(d.(string)), &r)
	return r, err
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
