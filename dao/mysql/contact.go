package mysql

import (
	"ForumSystem/models"
	"errors"
	"time"
)

type ContactService struct {
}

// 查找好友
func (mysql *ContactService) SearchFirend(userId int64) []models.User {
	conconts := make([]models.Contact, 0)
	objIds := make([]int64, 0)
	DbEngin.Where("ownerid = ? and cate = ?", userId, models.CONCAT_CATE_USER).Find(&conconts)
	for _, v := range conconts {
		objIds = append(objIds, v.Dstobj)
	}
	coms := make([]models.User, 0)
	if len(objIds) == 0 {
		return coms
	}
	DbEngin.In("user_id", objIds).Find(&coms)
	return coms
}

// 查找用户的群
func (mysql *ContactService) SearchComunity(userId int64) []models.ChatCommunity {
	conconts := make([]models.Contact, 0)
	comIds := make([]int64, 0)

	DbEngin.Where("ownerid = ? and cate = ?", userId, models.CONCAT_CATE_USER).Find(&conconts)
	for _, v := range conconts {
		comIds = append(comIds, v.Dstobj)
	}
	coms := make([]models.ChatCommunity, 0)
	if len(comIds) == 0 {
		return coms
	}
	DbEngin.In("user_id", comIds).Find(&coms)
	return coms
}

// 获取用户全部群ID
func (mysql *ContactService) SearchComunityIds(userId int64) (comIds []int64) {
	conconts := make([]models.Contact, 0) //  存储数据库中查找的内容
	comIds = make([]int64, 0)

	DbEngin.Where("ownerid = ? and cate = ?", userId, models.CONCAT_CATE_USER).Find(&conconts)
	for _, v := range conconts {
		comIds = append(comIds, v.Dstobj)
	}
	return comIds
}

// 加群
func (mysql *ContactService) JoinCommunity(userId, comId int64) error {
	//  将传入数据用一个结构体变量cot接收
	cot := models.Contact{
		Ownerid: userId,
		Dstobj:  comId,
		Cate:    models.CONCAT_CATE_COMUNITY,
	}
	//  到数据库中查询有没有存在数据，有就已经加入群聊了，没有则将数据插入到数据库中去
	DbEngin.Get(&cot)
	if cot.Id == 0 {
		cot.Createat = time.Now()
		_, err := DbEngin.Insert(cot)
		return err
	} else {
		return nil
	}
}

// 建群
func (mysql *ContactService) CreateCommunity(comm models.ChatCommunity) (ret models.ChatCommunity, err error) {
	if len(comm.Name) == 0 {
		err = errors.New("缺少群名称")
		return ret, err
	}

	if comm.Ownerid == 0 {
		err = errors.New("请先登录")
		return ret, err
	}
	com := models.ChatCommunity{Ownerid: comm.Ownerid}
	num, err := DbEngin.Count(&com)

	if num > 5 {
		err = errors.New("一个用户最多只能创建五个群")
		return com, err
	} else {
		comm.Createat = time.Now()
		session := DbEngin.NewSession()
		session.Begin()
		_, err := session.InsertOne(&comm)
		if err != nil {
			session.Rollback()
			return com, err
		}
		_, err = session.InsertOne(
			models.Contact{
				Ownerid:  comm.Ownerid,
				Dstobj:   comm.Id,
				Cate:     models.CONCAT_CATE_COMUNITY,
				Createat: time.Now(),
			})
		if err != nil {
			session.Rollback()
		} else {
			session.Commit()
		}
		return com, err
	}
}

// 添加好友
func (mysql *ContactService) AddFirend(userId, dstid int64) error {
	if userId == dstid {
		return errors.New("不能添加自己为好友!")
	}
	//  判断是否已经加了好友
	tmp := models.Contact{}
	//  查询数据库
	//  条件的链式存储
	DbEngin.Where("ownerid = ?", userId).
		And("dstid = ?", dstid).
		And("cate = ?", models.CONCAT_CATE_USER).Get(&tmp)
	//  获得一条记录
	//  如果存在记录说明已经是好友了不加
	if tmp.Id > 0 {
		return errors.New("该用户已经被添加过啦")
	}
	//  事务
	seesion := DbEngin.NewSession()
	seesion.Begin()
	//  插自己的
	_, e2 := seesion.InsertOne(models.Contact{
		Ownerid:  userId,
		Dstobj:   dstid,
		Cate:     models.CONCAT_CATE_USER,
		Createat: time.Now(),
	})
	//  插对方的
	_, e3 := seesion.InsertOne(models.Contact{
		Ownerid:  dstid,
		Dstobj:   userId,
		Cate:     models.CONCAT_CATE_USER,
		Createat: time.Now(),
	})
	//  没有错误
	if e2 == nil && e3 == nil {
		//  提交
		seesion.Commit()
		return nil
	} else {
		seesion.Rollback()
		if e2 != nil {
			return e2
		} else {
			return e3
		}
	}
}
