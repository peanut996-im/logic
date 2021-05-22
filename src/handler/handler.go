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
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		logger.Error("json unmarshal error. err: %v", err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	user, err := api.CheckToken(json["token"].(string))
	if db.IsNotExistError(err) {
		// token expired
		c.JSON(http.StatusOK, api.TokenInvaildResp)
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(user))
}

func LoadInitData(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		logger.Error("json unmarshal error. err: %v", err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	uid := json["user_id"].(string)
	// get user info
	user, err := model.GetUserByUID(uid)
	if nil != err {
		if db.IsNoDocumentError(err) {
			c.JSON(http.StatusOK, api.ResourceNotFoundResp)
			return
		}
		logger.Error("Logic.LoadInitData get Userinfo err: %v", err)
		c.JSON(http.StatusOK, api.NewHttpInnerErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, api.NewSuccessResponse(struct {
		User    *model.User     `json:"user_info"`
		Friends []*model.Friend `json:"friends_list"`
		Rooms   []*model.Room   `json:"room_list"`
	}{
		user,
		nil,
		nil,
	}))
}
