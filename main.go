package main

import (
	"go-tiny-url/app"
	"go-tiny-url/storage"
)

func main() {
	a := &app.App{}
	env := storage.NewEnv()
	a.Initialize(env)
	a.Run("0.0.0.0:8000")
}