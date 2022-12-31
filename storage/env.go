package storage

import (
	"context"
	"log"
	"os"
	"strconv"
)

type Env struct {
	St Storage
}

func NewEnv() *Env {
	addr := os.Getenv("TINYURL_APP_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6479"
	}
	passwd := os.Getenv("TINYURL_APP_REDIS_PWD")
	if passwd == "" {
		passwd = ""
	}

	dbSto := os.Getenv("TINYURL_APP_REDIS_DB")
	if dbSto == "" {
		dbSto = "0"
	}

	db, err := strconv.Atoi(dbSto)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	r := NewRedisCli(ctx, addr, passwd, db)
	log.Printf("connect to redis (addr: %s db:%d)", addr, db)
	return &Env{St: r}
}