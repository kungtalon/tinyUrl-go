package main

import (
	"flag"
	"go-tiny-url/app"
	"go-tiny-url/gql"
	"go-tiny-url/storage"
	"log"
)

func main() {
	useGql := flag.Bool("gql", false, "boolean: whether use graphql style api")
	flag.Parse()

	env := storage.NewEnv()
	if *useGql {
		log.Println("Using GQL")
		a := &gql.App{}
		a.Initialize(env)
		a.Run("0.0.0.0:8000")
	} else {
		a := &app.App{}
		a.Initialize(env)
		a.Run("0.0.0.0:8000")
	}
}	