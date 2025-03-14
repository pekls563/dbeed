package handler

import (
	"bigEventProject/account_srv/account_srv_params"
	"bigEventProject/mweb/articleCategory_srv_params"
	"bigEventProject/mweb/jwt_op"
	"bigEventProject/mweb/myredis"
	"bigEventProject/mweb/req"
	"bigEventProject/mweb/res"
	"bigEventProject/rpcProject/dclient"
	"bigEventProject/webProject/miniweb"
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/dgrijalva/jwt-go"
	"log"

	"net/http"
	"strconv"
	"time"
)

var registryAddr = "http://localhost:9091/krpc_/registry"

var AccountClient *dclient.XClient

var CategoryArticleClient *dclient.XClient

func init() {
	//err := initrpcClient()
	//if err != nil {
	//	panic(err)
	//}

	initrpcClient()

	fmt.Println(time.Now().String() + "初始化rpc客户端成功-------------------------------------------------------------------")

}

func initrpcClient() {

	d1 := dclient.NewKRegistryDiscovery(registryAddr, 0, "account_srv") //默认10秒

	AccountClient = dclient.NewXClient(d1, dclient.RoundRobinSelect, nil)

	d2 := dclient.NewKRegistryDiscovery(registryAddr, 0, "articleCategory_srv") //默认10秒

	CategoryArticleClient = dclient.NewXClient(d2, dclient.RoundRobinSelect, nil)

}

//func HealthHandler(c *gin.Context) {
//	c.JSON(http.StatusOK, gin.H{
//		"msg": "ok",
//	})
//}

//---------------------------------------------------/user 开始

func UserRegisterService(c *miniweb.Context) {
	var user req.UserRegisterServiceReq
	err := c.ShouldBindJSON(&user)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}
	//_, err = AccountClient.UserRegister(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_account.UserRegisterReq{
	//		UserName: user.UserName,
	//		Password: user.Password,
	//	})

	//err = xc.Call(ctx, serviceMethod, args, &reply)
	var empty account_srv_params.Empty

	err = AccountClient.Call(context.Background(), "AccountServer.UserRegister", &account_srv_params.UserRegisterReq{
		UserName: user.UserName,
		Password: user.Password,
	}, &empty)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "注册失败," + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})

}

func UserLoginService(c *miniweb.Context) {

	var userLoginServiceReq req.UserLoginServiceReq
	//var userLoginServiceRes res.UserLoginServiceRes
	err := c.ShouldBindJSON(&userLoginServiceReq)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	//redis获取图片验证码
	val, err := myredis.Redisclient.Get(context.Background(), userLoginServiceReq.VerifyStr).Result()
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "验证码过期",
		})
		return
	}

	if val != userLoginServiceReq.VerifyNumber {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "验证码错误",
		})
		return
	}

	var userLoginRes account_srv_params.UserLoginRes
	err = AccountClient.Call(context.Background(), "AccountServer.UserLogin", &account_srv_params.UserLoginReq{
		UserName: userLoginServiceReq.UserName,
		Password: userLoginServiceReq.Password,
	}, &userLoginRes)

	//userLoginRes, err := AccountClient.UserLogin(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_account.UserLoginReq{
	//		UserName: userLoginServiceReq.UserName,
	//		Password: userLoginServiceReq.Password,
	//	})
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "登录失败," + err.Error(),
		})
		return
	}

	//颁发Token
	j := jwt_op.NewJWT()
	//fmt.Println(string(*j.SigninKey))
	now := time.Now()

	claims := jwt_op.CustonClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Hour * 24 * 30).Unix(),
		},
		ID:          userLoginRes.Id,
		NickName:    "abcd",
		AuthorityId: int32(1234),
	}
	token, err := j.GenerateJWT(claims)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "token生成失败," + err.Error(),
		})
		return
	}
	//fmt.Println(token)

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
		"data": token,
	})
}

func UserInfoService(c *miniweb.Context) {

	idStr, _ := c.Get("claims")

	if idStr == nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}
	id := idStr.(int32)

	//userInfoRes, err := AccountClient.UserInfo(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_account.UserInfoReq{Id: id})
	var userInfoRes account_srv_params.UserInfoRes

	err := AccountClient.Call(context.Background(), "AccountServer.UserInfo",
		&account_srv_params.UserInfoReq{Id: id}, &userInfoRes)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "查询用户信息失败," + err.Error(),
		})
		return
	}
	//var userInfoServiceRes res.UserInfoServiceRes
	userInfoServiceRes := res.UserInfoServiceRes{
		Id:         int(userInfoRes.Id),
		UserName:   userInfoRes.UserName,
		NickName:   userInfoRes.NickName,
		Email:      userInfoRes.Email,
		UserPic:    userInfoRes.UserPic,
		CreateTime: "",
		UpdateTime: "",
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
		"data": userInfoServiceRes,
	})

}

func UserInfoUpdateService(c *miniweb.Context) {

	idStr, _ := c.Get("claims")

	if idStr == nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}
	id := idStr.(int32)

	var userInfoUpdateServiceReq req.UserInfoUpdateServiceReq
	err := c.ShouldBindJSON(&userInfoUpdateServiceReq)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}
	//_, err = AccountClient.UserInfoUpdate(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_account.UserInfoUpdateReq{
	//
	//		Id:       id,
	//		UserName: userInfoUpdateServiceReq.UserName,
	//		NickName: userInfoUpdateServiceReq.NickName,
	//		Email:    userInfoUpdateServiceReq.Email,
	//	})
	var empty account_srv_params.Empty
	err = AccountClient.Call(context.Background(), "AccountServer.UserInfoUpdate",
		&account_srv_params.UserInfoUpdateReq{
			Id:       id,
			UserName: userInfoUpdateServiceReq.UserName,
			NickName: userInfoUpdateServiceReq.NickName,
			Email:    userInfoUpdateServiceReq.Email,
		}, &empty)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "更新用户信息失败," + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})

}
func UserAvatarUpdateService(c *miniweb.Context) {
	idStr, _ := c.Get("claims")
	if idStr == nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	id := idStr.(int32)
	avatarUrlStr := c.Query("avatarUrl")

	if avatarUrlStr == "" {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	//_, err := AccountClient.UserAvatarUpdate(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_account.UserAvatarUpdateReq{
	//		Id:        id,
	//		AvatarUrl: avatarUrlStr,
	//	})
	var empty account_srv_params.Empty
	err := AccountClient.Call(context.Background(), "AccountServer.UserAvatarUpdate",
		&account_srv_params.UserAvatarUpdateReq{
			Id:        id,
			AvatarUrl: avatarUrlStr,
		}, &empty)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "更换头像失败",
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
		"data": "",
	})

}

//---------------------------------------------------/user 结束

//--------------------------------------------------/category 开始

func ArticleCategoryListService(c *miniweb.Context) {

	//articleCategoryListRes, err := CategoryArticleClient.ArticleCategoryList(context.WithValue(context.Background(), "ginContext", c),
	//	&emptypb.Empty{})
	var articleCategoryListRes articleCategory_srv_params.ArticleCategoryListRes
	err := CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleCategoryList",
		&articleCategory_srv_params.Empty{}, &articleCategoryListRes)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "获取分类列表错误," + err.Error(),
		})
		return
	}

	var category res.Category
	var categoryList []res.Category

	for _, item := range articleCategoryListRes.CategoryList {
		category.Id = int(item.Id)
		category.CategoryAlias = item.CategoryAlias
		category.CategoryName = item.CategoryName
		categoryList = append(categoryList, category)
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
		"data": categoryList,
	})

}

func ArticleCategoryAddService(c *miniweb.Context) {
	var articleCategoryAddServiceReq req.ArticleCategoryAddServiceReq
	err := c.ShouldBindJSON(&articleCategoryAddServiceReq)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	//_, err = CategoryArticleClient.ArticleCategoryAdd(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_category_article.ArticleCategoryAddReq{
	//		CategoryName:  articleCategoryAddServiceReq.CategoryName,
	//		CategoryAlias: articleCategoryAddServiceReq.CategoryAlias,
	//	})

	var empty articleCategory_srv_params.Empty
	err = CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleCategoryAdd",
		&articleCategory_srv_params.ArticleCategoryAddReq{
			CategoryName:  articleCategoryAddServiceReq.CategoryName,
			CategoryAlias: articleCategoryAddServiceReq.CategoryAlias,
		}, &empty)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "添加文章失败," + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})

}

func ArticleCategoryUpdateService(c *miniweb.Context) {
	var articleCategoryUpdateServiceReq req.ArticleCategoryUpdateServiceReq
	err := c.ShouldBindJSON(&articleCategoryUpdateServiceReq)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	//_, err = CategoryArticleClient.ArticleCategoryUpdate(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_category_article.ArticleCategoryUpdateReq{
	//		Id:            int32(articleCategoryUpdateServiceReq.Id),
	//		CategoryName:  articleCategoryUpdateServiceReq.CategoryName,
	//		CategoryAlias: articleCategoryUpdateServiceReq.CategoryAlias,
	//	})
	var empty articleCategory_srv_params.Empty
	err = CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleCategoryUpdate",
		&articleCategory_srv_params.ArticleCategoryUpdateReq{
			Id:            int32(articleCategoryUpdateServiceReq.Id),
			CategoryName:  articleCategoryUpdateServiceReq.CategoryName,
			CategoryAlias: articleCategoryUpdateServiceReq.CategoryAlias,
		}, &empty)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "更新文章分类错误",
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})

}

func ArticleCategoryDeleteService(c *miniweb.Context) {

	idStr := c.Query("id")

	id, _ := strconv.ParseInt(idStr, 10, 32)
	if id <= 0 {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "参数解析失败",
			"data": "",
		})
		return
	}

	//_, err := CategoryArticleClient.ArticleCategoryDelete(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_category_article.ArticleCategoryDeleteReq{Id: int32(id)})
	var empty articleCategory_srv_params.Empty
	err := CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleCategoryDelete",
		&articleCategory_srv_params.ArticleCategoryDeleteReq{Id: int32(id)}, &empty)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "删除文章分类错误," + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})
}

//--------------------------------------------------/category 结束

//--------------------------------------------------/article 开始

func ArticleListService(c *miniweb.Context) {
	pageNumStr := c.Query("pageNum")
	pageSizeStr := c.Query("pageSize")
	categoryIdStr := c.Query("categoryId")
	stateStr := c.Query("state")

	if pageNumStr == "" || pageSizeStr == "" || categoryIdStr == "" || stateStr == "" {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "参数解析失败",
		})
		return
	}

	pageNum, err := strconv.ParseInt(pageNumStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "参数解析失败",
		})
		return
	}
	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "参数解析失败",
		})
		return
	}
	categoryId, err := strconv.ParseInt(categoryIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "参数解析失败",
		})
		return
	}

	var articleListServiceRes res.ArticleListServiceRes
	//
	//articleListRes, err := CategoryArticleClient.ArticleList(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_category_article.ArticlelistReq{
	//		PageNum:    int32(pageNum),
	//		PageSize:   int32(pageSize),
	//		CategoryId: int32(categoryId),
	//		State:      stateStr,
	//	})
	var articleListRes articleCategory_srv_params.ArticleListRes
	err = CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleList",
		&articleCategory_srv_params.ArticlelistReq{
			PageNum:    int32(pageNum),
			PageSize:   int32(pageSize),
			CategoryId: int32(categoryId),
			State:      stateStr,
		}, &articleListRes)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "获取文章列表失败," + err.Error(),
		})
		return
	}

	articleListServiceRes.Total = int(articleListRes.Total)

	for _, item := range articleListRes.Items {
		article := res.Article{
			Id:         int(item.Id),
			Title:      item.Title,
			Content:    item.Content,
			CoverImg:   item.CoverImg,
			State:      item.State,
			CategoryId: int(item.CategoryId),
		}

		articleListServiceRes.Items = append(articleListServiceRes.Items, article)

	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
		//"data": articleListServiceRes,
		"total": articleListServiceRes.Total,
		"items": articleListServiceRes.Items,
	})

}

func ArticleAddService(c *miniweb.Context) {

	var articleAddServiceReq req.ArticleAddServiceReq
	err := c.ShouldBindJSON(&articleAddServiceReq)

	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "解析参数错误",
		})
		return
	}

	//rocketmqUrl := fmt.Sprintf("%s:%d", myconfig.AppConf.RocketmqConfig.Host, myconfig.AppConf.RocketmqConfig.Port)

	rocketmqUrl := "127.0.0.1:10909"

	//rocketmq模拟文章标题未过审-------------------------------------------------
	if articleAddServiceReq.Title == "weiguoshen" {
		p, err := rocketmq.NewProducer(
			producer.WithGroupName("wei_guo_shen1"),
			producer.WithNsResolver(primitive.NewPassthroughResolver([]string{rocketmqUrl})),
			producer.WithRetry(2))
		if err != nil {
			//log.Logger.Error(time.Now().String() + err.Error())
			log.Println(time.Now().String() + "初始化生产者错误" + err.Error())
		}
		err = p.Start()
		if err != nil {
			//log.Logger.Error(time.Now().String() + "生产者错误" + err.Error())
			log.Println(time.Now().String() + "启动生产者错误" + err.Error())

		}

		msg := &primitive.Message{
			Topic: "wei_guo_shen",
			Body:  []byte(articleAddServiceReq.Title),
		}

		//延迟发送,延迟10秒
		//实际程序调试是5秒？？？？？？？？？,原来是level=3
		msg.WithDelayTimeLevel(4)

		res, err := p.SendSync(context.Background(), msg)
		if err != nil {
			//log.Logger.Error(time.Now().String() + "发送消息错误" + err.Error())
			log.Println(time.Now().String() + "发送消息错误" + err.Error())
		} else {
			//log.Logger.Error(time.Now().String() + "发送消息成功" + res.String() + "---" + res.MsgID)
			log.Println(time.Now().String() + "发送消息成功" + res.String() + "---" + res.MsgID)

		}

		err = p.Shutdown()
		if err != nil {
			//log.Logger.Error(time.Now().String() + "生产者shutdown" + err.Error())
			log.Println(time.Now().String() + "生产者shutdown" + err.Error())

		}

	}

	///////////////////////////////////////////////////////////////////////
	//_, err = CategoryArticleClient.ArticleAdd(context.WithValue(context.Background(), "ginContext", c),
	//	&pb_category_article.ArticleAddReq{
	//		Title:      articleAddServiceReq.Title,
	//		Content:    articleAddServiceReq.Content,
	//		CoverImg:   articleAddServiceReq.CoverImg,
	//		State:      articleAddServiceReq.State,
	//		CategoryId: int32(articleAddServiceReq.CategoryId),
	//	})
	var empty articleCategory_srv_params.Empty
	err = CategoryArticleClient.Call(context.Background(), "ArticleCategoryServer.ArticleAdd",
		&articleCategory_srv_params.ArticleAddReq{
			Title:      articleAddServiceReq.Title,
			Content:    articleAddServiceReq.Content,
			CoverImg:   articleAddServiceReq.CoverImg,
			State:      articleAddServiceReq.State,
			CategoryId: int32(articleAddServiceReq.CategoryId),
		}, &empty)
	if err != nil {
		c.JSON(http.StatusOK, miniweb.H{
			"code": 1,
			"msg":  "添加文章失败," + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, miniweb.H{
		"code": 0,
		"msg":  "",
	})

}

//--------------------------------------------------/article 结束
