package res

/*type UserRegisterServiceRes struct {


}*/

type UserLoginServiceRes struct {
	Token string `json:"token"`
}

type UserInfoServiceRes struct {
	Id         int    `json:"id"`
	UserName   string `json:"username"`
	NickName   string `json:"nickname"`
	Email      string `json:"email"`
	UserPic    string `json:"userPic"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

/*type UserInfoUpdateServiceRes struct {


}*/

/*type UserAvatarUpdateServiceRes struct {

}*/

type Category struct {
	Id            int    `json:"id"`
	CategoryName  string `json:"categoryName"`
	CategoryAlias string `json:"categoryAlias"`
	CreateTime    string `json:"createTime"`
	UpdateTime    string `json:"updateTime"`
}

type ArticleCategoryListServiceRes struct {
	CategoryList []Category
}

/*type ArticleCategoryAddServiceRes struct {
}*/
/*type ArticleCategoryUpdateServiceRes struct {
}*/
/*type ArticleCategoryDeleteServiceRes struct {
}*/

type Article struct {
	Id         int    `json:"id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	CoverImg   string `json:"coverImg"`
	State      string `json:"state"`
	CategoryId int    `json:"categoryId"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

type ArticleListServiceRes struct {
	Total int       `json:"total"`
	Items []Article `json:"items"`
}

/*type ArticleAddServiceRes struct {


}*/
