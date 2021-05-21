package handler

import (
	"fmt"
	"framework/api"
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
	fmt.Println(json)
	uid := json["user_id"].(string)
	c.JSON(http.StatusOK, api.NewSuccessResponse(fmt.Sprintf("This is a init data for %v", uid)))
}
