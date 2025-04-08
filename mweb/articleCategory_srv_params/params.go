package articleCategory_srv_params


type Empty struct {
}

type Category struct {
	Id            int32
	CategoryName  string
	CategoryAlias string
}

type ArticleCategoryListRes struct {
	CategoryList []*Category
}

type ArticleCategoryAddReq struct {
	CategoryName  string
	CategoryAlias string
}

type ArticleCategoryUpdateReq struct {
	Id            int32
	CategoryName  string
	CategoryAlias string
}

type ArticleCategoryDeleteReq struct {
	Id int32
}

type ArticlelistReq struct {
	PageNum    int32
	PageSize   int32
	CategoryId int32
	State      string
}

type Article struct {
	Id         int32
	Title      string
	Content    string
	CoverImg   string
	State      string
	CategoryId int32
}

type ArticleListRes struct {
	Total int32
	Items []*Article
}

type ArticleAddReq struct {
	Title      string
	Content    string
	CoverImg   string
	State      string
	CategoryId int32
}
