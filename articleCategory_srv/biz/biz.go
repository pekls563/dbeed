package biz

import (
	"bigEventProject/articleCategory_srv/articleCategory_srv_params"
	"bigEventProject/articleCategory_srv/internal"
	"bigEventProject/articleCategory_srv/model"
	"bigEventProject/articleCategory_srv/myredis"
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

type ArticleCategoryServer struct {
}

func (a ArticleCategoryServer) ArticleCategoryList(empty *articleCategory_srv_params.Empty, res *articleCategory_srv_params.ArticleCategoryListRes) error {

	var categoryList []model.Category

	result := internal.DB.Scopes().Find(&categoryList)

	//----------------------------------------
	if result.Error != nil {
		return errors.New("服务端内部错误")
	}

	//	var categoryRes pb_category_article.Category
	for _, category := range categoryList {
		/*categoryRes.Id = int32(category.ID)
		categoryRes.CategoryName = category.CategoryName
		categoryRes.CategoryAlias = category.CategoryAlias
		categoryListRes.CategoryList = append(categoryListRes.CategoryList, &categoryRes)*/

		categoryRes := articleCategory_srv_params.Category{
			Id:            int32(category.ID),
			CategoryName:  category.CategoryName,
			CategoryAlias: category.CategoryAlias,
		}

		(*res).CategoryList = append((*res).CategoryList, &categoryRes)
	}
	return nil
}

func (a ArticleCategoryServer) ArticleCategoryAdd(req *articleCategory_srv_params.ArticleCategoryAddReq, res *articleCategory_srv_params.Empty) error {

	var category model.Category
	result := internal.DB.Where(&model.Category{CategoryName: req.CategoryName}).First(&category)
	if result.RowsAffected == 1 {
		return errors.New("类别已存在")
	}
	category.CategoryAlias = req.CategoryAlias
	category.CategoryName = req.CategoryName
	r := internal.DB.Create(&category)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}

	return nil

}

func (a ArticleCategoryServer) ArticleCategoryUpdate(req *articleCategory_srv_params.ArticleCategoryUpdateReq, res *articleCategory_srv_params.Empty) error {

	var category model.Category
	result := internal.DB.First(&category, req.Id)
	if result.RowsAffected == 0 {
		return errors.New("类别不存在")
	}
	category.CategoryAlias = req.CategoryAlias
	category.CategoryName = req.CategoryName
	r := internal.DB.Save(&category)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}
	return nil

}

func (a ArticleCategoryServer) ArticleCategoryDelete(req *articleCategory_srv_params.ArticleCategoryDeleteReq, res *articleCategory_srv_params.Empty) error {

	var category model.Category
	result := internal.DB.First(&category, req.Id)
	if result.RowsAffected == 0 {
		return errors.New("类别不存在")
	}
	r := internal.DB.Delete(&model.Category{}, req.Id)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}
	return nil
}

//分页逻辑判断
func MyPaging(pageNo, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if pageNo < 1 {
			pageNo = 1
		}
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize < 1:
			pageSize = 5

		}
		offset := (pageNo - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func (a ArticleCategoryServer) ArticleList(req *articleCategory_srv_params.ArticlelistReq, res *articleCategory_srv_params.ArticleListRes) error {

	var articleList []model.Article
	var articleList1 []model.Article

	//var articleRes pb_category_article.Article
	result1 := internal.DB.Where(&model.Article{
		CategoryId: int(req.CategoryId),
		State:      req.State,
	}).Scopes().Find(&articleList1)
	if result1.Error != nil {
		return errors.New("服务端内部错误")
	}

	result := internal.DB.Where(&model.Article{
		CategoryId: int(req.CategoryId),
		State:      req.State,
	}).Scopes(MyPaging(int(req.PageNum), int(req.PageSize))).Find(&articleList)
	if result.Error != nil {
		return errors.New("服务端内部错误")
	}
	(*res).Total = int32(result1.RowsAffected)
	for _, article := range articleList {

		articleRes := articleCategory_srv_params.Article{
			Id:         int32(article.ID),
			Title:      article.Title,
			Content:    article.Content,
			CoverImg:   article.CoverImg,
			State:      article.State,
			CategoryId: int32(article.CategoryId),
		}

		(*res).Items = append((*res).Items, &articleRes)

	}
	return nil

}

//使用redis分布式锁

func (a ArticleCategoryServer) ArticleAdd(req *articleCategory_srv_params.ArticleAddReq, res *articleCategory_srv_params.Empty) error {

	//抢锁,锁被某一个goroutine抢了以后，其他的就抢不到了，要等锁被释放,或者锁过期，其他goroutine才能再抢这个锁
	resp := myredis.Redisclient.SetNX(context.Background(), "ArticleAdd_lock", 1, time.Second*10)
	lockSuccess, err := resp.Result()

	if err != nil || !lockSuccess {
		//抢锁失败，业务结束
		//fmt.Println(err, "lock result: ", lockSuccess)
		return errors.New("请求过于频繁，请稍后再试")
	}

	//------------------------------------------------------------------------------------------------

	var article model.Article
	result := internal.DB.Where(&model.Article{Title: req.Title}).First(&article)
	if result.RowsAffected == 1 {
		return errors.New("该文章标题已存在")
	}

	article.Title = req.Title
	article.State = req.State
	article.CoverImg = req.CoverImg
	article.Content = req.Content
	article.CategoryId = int(req.CategoryId)
	r := internal.DB.Create(&article)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}

	return nil

}
