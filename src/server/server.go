package server

import (
	"framework/cfgargs"
	"framework/net"
	"framework/net/http"
	"logic/handler"
)

type Server struct{
	httpSrv *http.Server
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Init(cfg *cfgargs.SrvConfig){
	s.httpSrv = http.NewServer(cfg)
	s.httpSrv.AddNodeRoute(s.GetNodeRoute()...)
	go s.httpSrv.Serve(cfg) //nolint: errcheck
}

func(s *Server) GetNodeRoute() []*http.NodeRoute{
	var routers []*http.Route

	routers = append(routers, http.NewRoute(net.HTTP_METHOD_POST, "auth", handler.Auth))

	node := http.NewNodeRoute("", routers...)
	return []*http.NodeRoute{node}
}
