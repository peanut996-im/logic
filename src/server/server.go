package server

import (
	"encoding/json"
	"fmt"
	"framework/api"
	"framework/api/model"
	"framework/broker"
	"framework/cfgargs"
	"framework/logger"
	"framework/net/http"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg          *cfgargs.SrvConfig
	logicBroker  broker.LogicBroker
	httpSrv      *http.Server
	httpClient   *http.Client
	messageQueue chan *model.ChatMessage
	producer     *kafka.Producer
	consumer     *kafka.Consumer
	deliveryChan chan kafka.Event
}

func NewServer() *Server {
	return &Server{
		messageQueue: make(chan *model.ChatMessage, 5000),
		deliveryChan: make(chan kafka.Event, 5000),
	}
}

func (s *Server) Init(cfg *cfgargs.SrvConfig) {
	gin.DefaultWriter = logger.MultiWriter(logger.DefLogger().GetLogWriters()...)
	if cfg.Gate.Mode == "http" {
		s.logicBroker = broker.NewLogicBrokerHttp()
		s.logicBroker.Init(cfg)
	}
	s.logicBroker.Init(cfg)
	s.cfg = cfg
	s.httpClient = http.NewClient()
	s.httpSrv = http.NewServer()
	s.httpSrv.Init(cfg)
	s.MountRoute()

	// kafka

	c, _ := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": fmt.Sprintf("%v:%v", cfg.Kafka.Host, cfg.Kafka.Port),
		"group.id":          cfg.Kafka.Group,
		"auto.offset.reset": "earliest",
	})
	s.consumer = c

}

func (s *Server) Run() {
	go func() {
		s.Consume(s.ConsumeMessage)
	}()
	go s.logicBroker.Listen()
	//go s.httpSrv.Run()
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
		http.NewRoute(api.HTTPMethodPost, api.EventFindUser, s.FindUser),
		http.NewRoute(api.HTTPMethodPost, api.EventFindGroup, s.FindGroup),
		http.NewRoute(api.HTTPMethodPost, api.EventInviteFriend, s.InviteFriend),
		http.NewRoute(api.HTTPMethodPost, api.EventPullMessage, s.PullMessage),
		http.NewRoute(api.HTTPMethodPost, api.EventUpdateUser, s.UpdateUser),
		http.NewRoute(api.HTTPMethodPost, api.EventUpdateGroup, s.UpdateGroup),
	}
	node := http.NewNodeRoute(path, routers...)
	s.logicBroker.(*broker.LogicBrokerHttp).AddNodeRoute(node)
	s.httpSrv.AddNodeRoute(node)
}

func (s *Server) Produce(message *model.ChatMessage) {
	// MQ　producer

	j, _ := json.Marshal(message)

	s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &s.cfg.Kafka.Topic, Partition: kafka.PartitionAny},
		Value:          j,
	}, s.deliveryChan)

	e := <-s.deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		logger.Info("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		logger.Info("Delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}

	logger.Info("Logic.Produce: produce new message: [%+v]", *message)
	// s.messageQueue <- message
}

func (s *Server) Consume(consumerFunc func(message *model.ChatMessage)) {
	// TODO Replace By kafka
	go s.loopKafka(consumerFunc)
}

func (s *Server) loopKafka(consumerFunc func(message *model.ChatMessage)) {
	s.consumer.SubscribeTopics([]string{s.cfg.Kafka.Topic}, nil)
	for {
		msg, err := s.consumer.ReadMessage(-1)
		if nil == err {
			chatMsg := &model.ChatMessage{}
			json.Unmarshal(msg.Value, &chatMsg)
			consumerFunc(chatMsg)
		} else {
			logger.Error("Kafka Consumer error: %v (%v)", err, msg)
		}
	}
}
