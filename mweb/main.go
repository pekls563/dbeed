package main

import (
	"bigEventProject/mweb/handler"
	"bigEventProject/mweb/jwt_op"
	"bigEventProject/mweb/myredis"
	"bigEventProject/webProject/miniweb"
	"context"
	"fmt"

	"github.com/dchest/captcha"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//跨域
func CrossDomain(c *miniweb.Context) {

	method := c.Req.Method

	c.SetHeader("Access-Control-Allow-Origin", "*")
	c.SetHeader("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSrF-Token,Authorization,Token,x-token")
	c.SetHeader("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	c.SetHeader("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type")
	c.SetHeader("Access-Control-Allow-Credentials", "true")

	if method == "OPTIONS" {
		//c.AbortWithStatus(http.StatusNoContent)
		//c.JSON(http.StatusNoContent, miniweb.H{
		//	//不做处理
		//})
		c.AbortWithStatus(http.StatusNoContent)
	}
	c.Next()

}

//Token

func Token(c *miniweb.Context) {

	s := c.Req.URL.String()

	if s == "/health" || s == "/user/register" || s == "/user/login" || s == "/test" || s == "/getImage" || strings.HasPrefix(s, "/getImageName") {
		c.Next()
		return
	}

	token := c.Req.Header.Get("Authorization")
	if token == "" || len(token) == 0 {
		c.JSON(http.StatusUnauthorized, miniweb.H{
			"msg": "认证失败，需要登录",
		})
		c.Abort()
		return
	}
	fmt.Println("用户token为", token)
	j := jwt_op.NewJWT()
	parseToken, err := j.ParseToken(token)

	if err != nil {
		if err.Error() == jwt_op.TokenExpired {
			c.JSON(http.StatusUnauthorized, miniweb.H{
				"msg": jwt_op.TokenExpired,
			})
			c.Abort()
			return
		}
		c.JSON(http.StatusUnauthorized, miniweb.H{
			"msg": "认证失败，需要登录",
		})
		c.Abort()
		return
	}
	c.Set("claims", parseToken.ID)

	c.Next()

}

func init() {
	myredis.InitRedis()
}

func main() {

	r := miniweb.Default()

	r.Use(CrossDomain, Token)

	accountGroup := r.Group("/user")
	{
		accountGroup.POST("/register", handler.UserRegisterService)
		accountGroup.POST("/login", handler.UserLoginService)
		accountGroup.GET("/userInfo", handler.UserInfoService)
		accountGroup.PUT("/update", handler.UserInfoUpdateService)
		accountGroup.PATCH("/updateAvatar", handler.UserAvatarUpdateService)
		//accountGroup.PATCH("/updatePwd")

	}

	categoryGroup := r.Group("/category")
	{
		categoryGroup.GET("/list", handler.ArticleCategoryListService)
		categoryGroup.POST("/add", handler.ArticleCategoryAddService)
		categoryGroup.PUT("/update", handler.ArticleCategoryUpdateService)
		categoryGroup.DELETE("/delete", handler.ArticleCategoryDeleteService)
	}

	articleGroup := r.Group("/article")
	{
		articleGroup.GET("/list", handler.ArticleListService)
		articleGroup.POST("/add", handler.ArticleAddService)
	}

	r.GET("/test", func(c *miniweb.Context) {
		c.JSON(http.StatusOK, miniweb.H{
			"msg": "testok",
		})
	})

	//创建图片验证码
	r.GET("/getImage", func(c *miniweb.Context) {
		randUUID := uuid.New().String()

		fileName := fmt.Sprintf("mweb\\image\\%s.png", randUUID)

		f, err := os.Create(fileName)
		if err != nil {
			//log.Logger.Error("创建文件失败" + time.Now().String())
			log.Println("创建文件失败" + time.Now().String())
			c.JSON(http.StatusOK, miniweb.H{
				"code": 1,
				"msg":  "创建文件失败",
			})
			return
		}
		defer f.Close()
		var w io.WriterTo

		//生成6位数字验证码
		d := captcha.RandomDigits(captcha.DefaultLen)

		//由验证码生成图片
		w = captcha.NewImage("", d, captcha.StdWidth, captcha.StdHeight)
		_, err = w.WriteTo(f)
		if err != nil {
			//log.Logger.Error("GenCaptcha() 失败" + time.Now().String())
			log.Println("GenCaptcha() 失败" + time.Now().String())
			c.JSON(http.StatusOK, miniweb.H{
				"code": 1,
				"msg":  "GenCaptcha() 失败",
			})
			return
		}

		captcha := ""

		//将字节转换为数字
		for _, item := range d {
			captcha += fmt.Sprintf("%d", item)

		}

		//向redis数据库发送数据,保存120秒
		myredis.Redisclient.Set(context.Background(), randUUID, captcha, 120*time.Second)

		//baseurl2 := fmt.Sprintf("http://%s:%d/getImageName?imageName=", myconfig.AppConf.DashijianWebSrv.Host, myconfig.AppConf.DashijianWebSrv.Port)
		baseurl2 := fmt.Sprintf("http://%s:%d/getImageName?imageName=", "127.0.0.1", 9090)

		url2 := baseurl2 + randUUID + ".png"

		c.JSON(http.StatusOK, miniweb.H{
			"code": 0,
			//"url":       "http://127.0.0.1:9097/getImageName?imageName=" + randUUID + ".png",
			"url":       url2,
			"verifyStr": randUUID,
		})

	})

	//获取图片验证码.png文件
	r.GET("/getImageName", func(c *miniweb.Context) {

		imageName := c.Query("imageName")

		filePath := fmt.Sprintf("mweb\\image\\%s", imageName)
		file, _ := ioutil.ReadFile(filePath) //把要显示的图片读取到变量中

		c.SetHeader("Content-Type", "image/png")

		c.Data(200, file)
	})

	r.GET("/ttst", func(c *miniweb.Context) {
		c.JSON(http.StatusOK, miniweb.H{
			"ttst": "大事件测试成功",
		})
	})

	log.Println(time.Now().String() + "web启动成功")

	//web端口9090
	r.Run(":9090")

}
