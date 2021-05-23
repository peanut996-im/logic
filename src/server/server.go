package server

import (
	"framework/api"
	"framework/cfgargs"
	"framework/logger"
	"framework/net/http"
	"github.com/gin-gonic/gin"
	"logic/handler"
)

type Server struct {
	httpSrv *http.Server
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Init(cfg *cfgargs.SrvConfig) {
	gin.DefaultWriter = logger.MultiWriter(logger.DefLogger().GetLogWriters()...)
	s.httpSrv = http.NewServer(cfg)
	s.MountRoute()
	go s.httpSrv.Serve(cfg) //nolint: errcheck
}

func (s *Server) MountRoute() {
	path := ""
	routers := []*http.Route{
		// TODO: Mount routes
		http.NewRoute(api.HTTPMethodPost, api.EventAuth, handler.Auth),
		//http.NewRoute(api.HTTPMethodPost, api.EventAuth, handler.EventHandler(api.EventAuth)),
		http.NewRoute(api.HTTPMethodPost, api.EventLoad, handler.Load),
		//http.NewRoute(api.HTTPMethodPost, api.EventLoad, handler.EventHandler(api.EventLoad)),
		http.NewRoute(api.HTTPMethodPost, api.EventAddFriend, handler.AddFriend),
		http.NewRoute(api.HTTPMethodPost, api.EventDeleteFriend, handler.DeleteFriend),
		http.NewRoute(api.HTTPMethodPost, api.EventCreateGroup, handler.CreateGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventJoinGroup, handler.JoinGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventLeaveGroup, handler.LeaveGroup),
	}
	node := http.NewNodeRoute(path, routers...)
	s.httpSrv.AddNodeRoute(node)
}
