package handler

import (
	"framework/api"
	"framework/api/model"
	"framework/db"
	"framework/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Auth(c *gin.Context) {
	aR := &api.AuthRequest{}
	err := c.BindJSON(aR)
	if err != nil {
		logger.Error("Logic.Auth "+api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	user, err := api.CheckToken(aR.Token)
	if db.IsNotExistError(err) {
		// token expired
		c.JSON(http.StatusOK, api.TokenInvaildResp)
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(user))
}

func Load(c *gin.Context) {
	lR := &api.LoadRequest{}
	err := c.BindJSON(lR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	// get user info
	user, err := model.GetUserByUID(lR.UID)
	if nil != err {
		if db.IsNoDocumentError(err) {
			c.JSON(http.StatusOK, api.ResourceNotFoundResp)
			return
		}
		logger.Error("Logic.Load "+api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	friends,err := model.GetAllFriends(lR.UID)
	if err!=nil{
		logger.Error("Logic.Load "+api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	rooms,err := model.GetRoomsByUID(lR.UID)
	if err!=nil{
		logger.Error("Logic.Load "+api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(struct {
		User    *model.User     `json:"userInfo"`
		Friends []string `json:"friendList"`
		Rooms   []string  `json:"roomList"`
	}{
		user,
		friends,
		rooms,
	}))
}

func AddFriend(c *gin.Context) {
	fR := &api.FriendRequest{}
	err := c.BindJSON(fR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.AddNewFriend(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func DeleteFriend(c *gin.Context) {
	fR := &api.FriendRequest{}
	err := c.BindJSON(fR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.DeleteFriend(fR.FriendA, fR.FriendB)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func CreateGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.CreateGroup(gR.GroupName, gR.GroupAdmin)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func JoinGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.CreateGroupUser(gR.GroupID, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

func LeaveGroup(c *gin.Context) {
	gR := &api.GroupRequest{}
	err := c.BindJSON(gR)
	if err != nil {
		logger.Error(api.UnmarshalJsonError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	err = model.DeleteGroupUser(gR.GroupID, gR.UID)
	if err != nil {
		logger.Error(api.MongoDBError, err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(nil))
}

// refactor
//func EventHandler(event string) func(c *gin.Context){
//	return func(c *gin.Context){
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
