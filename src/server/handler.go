package server

import (
	"fmt"
	"framework/api"
	"framework/api/model"
	"framework/db"
	"framework/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

func (s *Server) Chat(c *gin.Context) {
	cR := &api.ChatRequest{}
	err := c.BindJSON(cR)
	if err != nil {
		logger.Error("Logic.Auth "+api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	msg := model.ChatMessageFrom(cR.From, cR.To, cR.Content.(string), cR.Type)
	// TODO replace for MQ
	go model.InsertChatMessage(msg)
	go s.PushChatMessage(msg)
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) PushChatMessage(message *model.ChatMessage) {
	ch := make(chan struct{}, 0)
	defer close(ch)
	pCR := &api.PushChatRequest{
		Message: message,
	}
	roomID := message.To
	room, err := model.GetRoomByID(roomID)
	if err != nil {
		logger.Error("Logic.PushChat no such room: %v", roomID)
		return
	}
	targets := []string{}
	if room.OneToOne {
		logger.Debug("Logic.Chat Push Friend Message")
		//single
		targets, err = model.GetFriendsByRoomID(room.RoomID)
		if err != nil {
			return
		}

	} else {
		//group
		targets, err = model.GetUserIDsByGroupID(message.To)
		if err != nil {
			logger.Error("Logic.PushChat Get Group Users err: %v", err)
			return
		}
	}
	for _, target := range targets {
		pCR.Target = target
		go s.SendToTarget(ch, target, pCR)
		<-ch
	}
}

func (s *Server) SendToTarget(ch chan struct{}, target string, request *api.PushChatRequest) {
	logger.Debug("Logic.SendToTarget target: %v", target)
	url, err := s.GetChatUrlFromScene(target)
	if err != nil {
		logger.Error("Logic.PushChat GetGateUrl err: %v", err)
		return
	}
	go s.httpClient.GetGoReq().Post(url).Send(request).End()
	ch <- struct{}{}
}

func (s *Server) GetGateAddrFromScene(scene string) (string, error) {
	//TODO for cluster fix get from server
	return fmt.Sprintf("%v:%v", s.cfg.Gate.Host, s.cfg.Gate.Port), nil
}

func (s *Server) GetChatUrlFromScene(scene string) (string, error) {
	addr, err := s.GetGateAddrFromScene(scene)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%v/%v", addr, api.EventChat), nil
}

func (s *Server) GetUserInfo(c *gin.Context) {
	uR := &api.UserRequest{}
	err := c.BindJSON(uR)
	if err != nil {
		logger.Error("Logic.Auth "+api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	user, err := model.GetUserByUID(uR.UID)
	if err != nil {
		logger.Error("Logic.GetUserInfo "+api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(user))
}

func (s *Server) Auth(c *gin.Context) {
	aR := &api.AuthRequest{}
	err := c.BindJSON(aR)
	if err != nil {
		logger.Error("Logic.Auth "+api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	user, err := api.CheckToken(aR.Token)
	if db.IsNotExistError(err) {
		// token expired
		c.AbortWithStatusJSON(http.StatusOK, api.TokenInvaildResp)
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(user))
}

func (s *Server) Load(c *gin.Context) {
	lR := &api.LoadRequest{}
	err := c.BindJSON(lR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	var wg sync.WaitGroup
	// 防止waitgroup.wait()最先执行
	wg.Add(1)
	uCh, fCh, rCh := make(chan *model.User, 0), make(chan []*model.FriendData, 0), make(chan []*model.Room, 0)
	//gCh, gUCh := make(chan []*model.GroupData, 0), make(chan []*model.User, 0)
	gCh := make(chan []*model.GroupData, 0)
	done := make(chan struct{})
	defer close(uCh)
	defer close(fCh)
	defer close(rCh)
	defer close(gCh)
	//defer close(gUCh)
	// get user info
	go func() {
		wg.Add(1)
		defer wg.Done()
		user, err := model.GetUserByUID(lR.UID)
		if nil != err {
			uCh <- nil
			return
		}
		uCh <- user
	}()

	go func() {
		// friends
		wg.Add(1)
		defer wg.Done()
		friends, err := model.GetFriendDatasByUID(lR.UID)
		if err != nil {
			fCh <- nil
			return
		}
		fCh <- friends
	}()

	go func() {
		// rooms
		wg.Add(1)
		defer wg.Done()
		rooms, err := model.GetRoomsByUID(lR.UID)
		if err != nil {
			rCh <- nil
			return
		}
		rCh <- rooms
	}()

	go func() {
		// rooms
		wg.Add(1)
		defer wg.Done()
		groups, err := model.GetGroupDatasByUID(lR.UID)
		if err != nil {
			gCh <- nil
			//gUCh <- nil
			return
		}
		gCh <- groups
		//groupUsers, err := model.GetUsersByGroups(groups...)
		//if nil != err {
		//	gUCh <- nil
		//	return
		//}
		//gUCh <- groupUsers
		// Done 抵消一开始的add(1) 保证这里的执行完毕
		wg.Done()
	}()
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()
	user, friends, rooms, groups, groupUsers := &model.User{}, []*model.FriendData{}, []*model.Room{}, []*model.GroupData{}, []*model.User{}
Loop:
	for {
		select {
		case <-done:
			break Loop
		case u := <-uCh:
			user = u
		case f := <-fCh:
			friends = f
		case r := <-rCh:
			rooms = r
		case g := <-gCh:
			groups = g
		//case gU := <-gUCh:
		//	groupUsers = gU
		case <-time.After(1 * time.Second):
			break Loop
		}
	}
	if user == nil || friends == nil || rooms == nil || groups == nil || groupUsers == nil {
		logger.Error("Logic.Load "+api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, api.NewSuccessResponse(struct {
		User    *model.User         `json:"user"`
		Friends []*model.FriendData `json:"friends"`
		Groups  []*model.GroupData  `json:"groups"`
		//Rooms      []*model.Room             `json:"rooms"`
		//GroupUsers []*model.User             `json:"groupUsers"`
	}{
		user,
		friends,
		//rooms,
		groups,
		//groupUsers,
	}))
}

func (s *Server) AddFriend(c *gin.Context) {
	fR := &api.FriendRequest{}
	err := c.BindJSON(fR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.AddNewFriend(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) DeleteFriend(c *gin.Context) {
	fR := &api.FriendRequest{}
	err := c.BindJSON(fR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.DeleteFriend(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) CreateGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.CreateGroup(gR.GroupName, gR.GroupAdmin)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) JoinGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.CreateGroupUser(gR.GroupID, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) LeaveGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.DeleteGroupUser(gR.GroupID, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

// refactor
//func(s *Server)  EventHandler(event string) func(s *Server) (c *gin.Context){
//	return func(s *Server) (c *gin.Context){
//		var (
//			data interface{}
//			err error
//		)
//		switch event {
//		case api.EventAuth:
//			data = &api.AuthRequest{}
//		case api.EventLoad:
//			data = &api.LoadRequest{}
//		case api.EventAddFriend,api.EventDeleteFriend:
//			data = &api.FriendRequest{}
//		case api.EventCreateGroup, api.EventJoinGroup,api.EventLeaveGroup:
//			data = &api.GroupRequest{}
//		default:
//			c.JSON(http.StatusNotFound,nil)
//			return
//		}
//		err = c.BindJSON(data)
//		if err != nil {
//			logger.Error(fmt.Sprintf("Logic.Handler %v ",event)+api.UnmarshalJsonError,err)
//			c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//			return
//		}
//		switch event {
//		case api.EventAuth:
//			user, err := api.CheckToken(data.(*api.AuthRequest).Token)
//			if db.IsNotExistError(err) {
//				// token expired
//				c.JSON(http.StatusOK, api.TokenInvaildResp)
//				return
//			}
//			c.JSON(http.StatusOK, api.NewSuccessResponse(user))
//			return
//		case api.EventLoad:
//			user, err := model.GetUserByUID(data.(*api.LoadRequest).UID)
//			if nil != err {
//				if db.IsNoDocumentError(err) {
//					c.JSON(http.StatusOK, api.ResourceNotFoundResp)
//					return
//				}
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//			c.JSON(http.StatusOK, api.NewSuccessResponse(struct {
//				User    *model.User     `json:"userInfo"`
//				Friends []*model.Friend `json:"friendList"`
//				Rooms   []*model.Room   `json:"roomList"`
//			}{
//				user,
//				nil,
//				nil,
//			}))
//			return
//		case api.EventAddFriend:
//			err = model.AddNewFriend(data.(*api.FriendRequest).FriendA, data.(*api.FriendRequest).FriendB)
//			if err != nil {
//				logger.Error(api.MongoDBError,err)
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//		case api.EventDeleteFriend:
//			err = model.DeleteFriend(data.(*api.FriendRequest).FriendA, data.(*api.FriendRequest).FriendB)
//			if err != nil {
//				logger.Error(api.MongoDBError,err)
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//		case api.EventCreateGroup:
//			err = model.CreateGroup(data.(*api.GroupRequest).GroupName, data.(*api.GroupRequest).GroupID)
//			if err != nil {
//				logger.Error(api.MongoDBError,err)
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//		case api.EventJoinGroup:
//			err = model.CreateGroupUser(data.(*api.GroupRequest).GroupID,data.(*api.GroupRequest).UID)
//			if err != nil {
//				logger.Error(api.MongoDBError,err)
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//		case api.EventLeaveGroup:
//			err = model.DeleteGroupUser(data.(*api.GroupRequest).GroupID,data.(*api.GroupRequest).UID)
//			if err != nil {
//				logger.Error(api.MongoDBError,err)
//				c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
//				return
//			}
//		default:
//			c.JSON(http.StatusNotFound,nil)
//			return
//		}
//		c.JSON(http.StatusOK,api.NewSuccessResponse(nil))
//	}
//}
