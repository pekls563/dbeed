package main

import (
	"bigEventProject/webProject/miniweb"
	"fmt"
	"net/http"
)

type Article struct {
	ArticleName string `json:"article_name"`
	ArticleNo   int    `json:"article_no"`
}

func main() {

	r := miniweb.Default()

	r.POST("/ttst", func(c *miniweb.Context) {
		var article Article
		err := c.ShouldBindJSON(&article)
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusOK, miniweb.H{
				"msg": "解析参数错误",
			})
			return
		}

		fmt.Println(article.ArticleName, "+", article.ArticleNo)

		c.JSON(http.StatusOK, miniweb.H{
			"ttst": "测试成功",
		})
	})

	r.Run("localhost:9093")

}
