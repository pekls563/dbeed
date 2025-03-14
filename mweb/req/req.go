package req

type Image struct {
	ImageName string `json:"imageName"`
}

type UserRegisterServiceReq struct {
	UserName   string `json:"username"`
	Password   string `json:"password"`
	RePassword string `json:"rePassword"`
}

type UserLoginServiceReq struct {
	UserName     string `json:"username"`
	Password     string `json:"password"`
	VerifyNumber string `json:"verifyNumber"`
	VerifyStr    string `json:"verifyStr"`
}

type UserInfoServiceReq struct {
	Id string `json:"id"`
}
type UserInfoUpdateServiceReq struct {
	Id       int    `json:"id"`
	UserName string `json:"username"`
	NickName string `json:"nickname"`
	Email    string `json:"email"`
}
type UserAvatarUpdateServiceReq struct {
	AvatarUrl string `json:"avatarUrl"`
}

/*type ArticleCategoryListServiceReq struct {
}*/
type ArticleCategoryAddServiceReq struct {
	CategoryName  string `json:"categoryName"`
	CategoryAlias string `json:"categoryAlias"`
}
type ArticleCategoryUpdateServiceReq struct {
	Id            int    `json:"id"`
	CategoryName  string `json:"categoryName"`
	CategoryAlias string `json:"categoryAlias"`
}

/*type ArticleCategoryDeleteServiceReq struct {
}*/

/*type ArticleListServiceReq struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	CoverImg   string `json:"coverImg"`
	State      string `json:"state"`
	CategoryId string `json:"categoryId"`
}*/

type ArticleAddServiceReq struct {
	Title      string `json:"title"`
	Content    string `json:"content" `
	CoverImg   string `json:"coverImg"`
	State      string `json:"state"`
	CategoryId int    `json:"categoryId"`
}
