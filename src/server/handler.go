package server

import (
	"framework/api"
	"framework/api/model"
	"framework/db"
	"framework/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

func (s *Server) Chat(c *gin.Context) {
	cR := &api.ChatRequest{}
	err := c.BindJSON(cR)
	if err != nil {
		logger.Error("Logic.Auth "+api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	msg := model.ChatMessageFrom(cR.From, cR.To, cR.Content, cR.Type, cR.Height, cR.Width, cR.Size, cR.FileName)
	// TODO replace for MQ
	go model.InsertChatMessage(msg)
	go s.PushChatMessage(msg)
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
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
	var lock sync.RWMutex
	user, friends, groups := &model.User{}, []*model.FriendData{}, []*model.GroupData{}
	errs := make([]error, 0)

	wg.Add(1)
	go func(uid string) {
		defer wg.Done()
		u, err := model.GetUserByUID(uid)
		if nil != err {
			lock.Lock()
			errs = append(errs, err)
			lock.Unlock()
			return
		}
		user = u
	}(lR.UID)

	wg.Add(1)
	go func(uid string) {
		// friends
		defer wg.Done()
		fs, err := model.GetFriendDatasByUID(uid)
		if err != nil {
			lock.Lock()
			errs = append(errs, err)
			lock.Unlock()
			return
		}
		friends = fs
	}(lR.UID)

	wg.Add(1)
	go func(uid string) {
		// group
		defer wg.Done()
		gs, err := model.GetGroupDatasByUID(uid)
		if err != nil {
			lock.Lock()
			errs = append(errs, err)
			lock.Unlock()
			return
		}
		groups = gs
	}(lR.UID)
	wg.Wait()

	if len(errs) > 0 {
		logger.Error("Logic.Load err: %v", errs[0])
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, api.NewSuccessResponse(struct {
		User    *model.User         `json:"user"`
		Friends []*model.FriendData `json:"friends"`
		Groups  []*model.GroupData  `json:"groups"`
	}{
		user,
		friends,
		groups,
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
	friendData, err := model.GetFriendDataByIDs(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(friendData))
}

func (s *Server) DeleteFriend(c *gin.Context) {
	fR := &api.FriendRequest{}
	err := c.BindJSON(fR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	friend, err := model.DeleteFriend(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(friend))
}

func (s *Server) CreateGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	group, err := model.CreateGroup(gR.GroupName, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	// 返回数据
	groupData, err := model.GetGroupDataByGroupID(group.GroupID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(groupData))
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
	groupData, err := model.GetGroupDataByGroupID(gR.GroupID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(groupData))
}

func (s *Server) LeaveGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	gUser, err := model.DeleteGroupUser(gR.GroupID, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(gUser))
}

func (s *Server) FindUser(c *gin.Context) {
	fUR := &api.FindRequest{}
	err := c.BindJSON(fUR)
	if nil != err {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	users, err := model.FindUsersByAccount(fUR.Account)
	if nil != err {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(users))
}

func (s *Server) FindGroup(c *gin.Context) {
	fUR := &api.FindRequest{}
	err := c.BindJSON(fUR)
	if nil != err {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	groups, err := model.FindGroupsByGroupName(fUR.GroupName)
	if nil != err {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(groups))
}

func (s *Server) InviteFriend(c *gin.Context) {
	iR := &api.InviteRequest{}
	err := c.BindJSON(iR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = s.InviteFriendsToGroup(iR.Friends, iR.GroupID)
	if err != nil {
		logger.Error("InviteFriendsToGroup err: %v", err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func (s *Server) PullMessage(c *gin.Context) {
	pR := &api.PullRequest{}
	err := c.BindJSON(pR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	messages, err := s.PullMessageByPage(pR.UID, pR.FriendID, pR.GroupID, pR.Current, pR.PageSize)
	if err != nil {
		logger.Error("PullMessage err: %v", err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(messages))
}

func (s *Server) UpdateUser(c *gin.Context) {
	uR := &api.UpdateRequest{}
	err := c.BindJSON(uR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	user, err := s.UpdateUserInfo(uR.UID, uR.Account, uR.Password, uR.Avatar)
	if nil != err {
		logger.Error("UpdateUserInfo err: %v", err)
		c.AbortWithStatusJSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(user))
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
