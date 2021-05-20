package handler

import (
	"framework/api"
	"framework/logger"
	"framework/net"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Auth(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		logger.Error("json unmarshal error. err: %v", err)
		c.JSON(http.StatusOK, net.NewHttpInnerErrorResp(err))
		return
	}
	user, err := api.CheckToken(json["token"].(string))
	c.JSON(http.StatusOK,net.NewSuccessResponse(user))
}
