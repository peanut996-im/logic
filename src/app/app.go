package app

import (
	"framework/cfgargs"
	"framework/db"
	"framework/logger"
	"logic/server"
	"sync"
)


var (
	once sync.Once
	app  *App
)

type App struct {
	cfg *cfgargs.SrvConfig
	srv *server.Server
}

func GetApp() *App {
	once.Do(func() {
		a := &App{}
		app = a
	})
	return app
}

func (a *App) Init(cfg *cfgargs.SrvConfig) {
	db.InitRedisClient(cfg)
	err := db.InitMongoClient(cfg)
	if err != nil {
		logger.Fatal("init mongo db err: %v", err)
		return
	}

	a.srv = server.NewServer()
	a.srv.Init(cfg)

}
