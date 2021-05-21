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
	s.httpSrv.AddNodeRoute(s.GetNodeRoute()...)
	go s.httpSrv.Serve(cfg) //nolint: errcheck
}

func (s *Server) GetNodeRoute() []*http.NodeRoute {
	routers := []*http.Route{
		// TODO: Mount routes
		http.NewRoute(api.HTTP_METHOD_POST, "auth", handler.Auth),
		http.NewRoute(api.HTTP_METHOD_POST, "load", handler.LoadInitData),
	}

	node := http.NewNodeRoute("", routers...)
	return []*http.NodeRoute{node}
}
