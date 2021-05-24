package server

import (
	"framework/api"
	"framework/cfgargs"
	"framework/logger"
	"framework/net/http"
	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg        *cfgargs.SrvConfig
	httpSrv    *http.Server
	httpClient *http.Client
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Init(cfg *cfgargs.SrvConfig) {
	gin.DefaultWriter = logger.MultiWriter(logger.DefLogger().GetLogWriters()...)
	s.cfg = cfg
	s.httpClient = http.NewClient()
	s.httpSrv = http.NewServer()
	s.httpSrv.Init(cfg)
	s.MountRoute()
}
func (s *Server) Run() {
	go s.httpSrv.Run()
}

func (s *Server) MountRoute() {
	path := ""
	routers := []*http.Route{
		// TODO: Mount routes
		http.NewRoute(api.HTTPMethodPost, api.EventChat, s.Chat),
		http.NewRoute(api.HTTPMethodPost, api.EventAuth, s.Auth),
		http.NewRoute(api.HTTPMethodPost, api.EventLoad, s.Load),
		http.NewRoute(api.HTTPMethodPost, api.EventAddFriend, s.AddFriend),
		http.NewRoute(api.HTTPMethodPost, api.EventDeleteFriend, s.DeleteFriend),
		http.NewRoute(api.HTTPMethodPost, api.EventCreateGroup, s.CreateGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventJoinGroup, s.JoinGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventLeaveGroup, s.LeaveGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventGetUserInfo, s.GetUserInfo),
	}
	node := http.NewNodeRoute(path, routers...)
	s.httpSrv.AddNodeRoute(node)
}
