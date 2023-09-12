package controller

import (
	"ForumSystem/logic"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

func Chat(c *gin.Context) {
	//  校验参数
	id := c.Query("id") //  Query返回是String类型
	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		zap.L().Error("Query fail")
		return
	}
	err = logic.Chat(userID, c)
	if err != nil {
		zap.L().Error("Chat has err", zap.Error(err))
		return
	}
	ResponseSuccess(c, nil)
}
