package controller

import (
	"ForumSystem/args"
	"ForumSystem/dao/mysql"
	"ForumSystem/logic"
	"ForumSystem/models"
	"github.com/gin-gonic/gin"
)

var contactService mysql.ContactService

// 加载个人好友
func LoadFriend(c *gin.Context) {
	var arg args.ContactArg
	c.Bind(&arg)

	users := contactService.SearchFirend(arg.Userid)
	RespOkList(c.Writer, users, len(users))
}

// 加载用户的群
func LoadCommunity(c *gin.Context) {
	var arg args.ContactArg
	c.Bind(&arg)
	communitys := contactService.SearchComunity(arg.Userid)
	RespOkList(c.Writer, communitys, len(communitys))
}

// 创建群
func CreateCommunity(c *gin.Context) {
	var arg models.ChatCommunity
	c.Bind(&arg)
	com, err := contactService.CreateCommunity(arg)
	if err != nil {
		RespFail(c.Writer, err.Error())
	} else {
		RespOk(c.Writer, com, "")
	}
}

// 添加好友
func AddFriend(c *gin.Context) {
	var arg args.ContactArg
	c.Bind(&arg)
	err := contactService.AddFirend(arg.Userid, arg.Dstid)
	if err != nil {
		RespFail(c.Writer, err.Error())
	} else {
		RespOk(c.Writer, nil, "好友添加成功")
	}
}

// 加入群
func JoinCommunityc(c *gin.Context) {
	var arg args.ContactArg
	c.Bind(&arg)
	err := contactService.JoinCommunity(arg.Userid, arg.Dstid)
	//  刷新用户的群组消息
	logic.AddGroupId(arg.Userid, arg.Dstid)
	if err != nil {
		RespFail(c.Writer, err.Error())
	} else {
		RespOk(c.Writer, nil, "")
	}
}
