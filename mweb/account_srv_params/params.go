package account_srv_params

type UserRegisterReq struct {
	UserName string
	Password string
}

type Empty struct {
}

type UserLoginReq struct {
	UserName string
	Password string
}

type UserLoginRes struct {
	Id int32
}

type UserInfoReq struct {
	Id int32
}

type UserInfoRes struct {
	Id       int32
	UserName string
	NickName string
	Email    string
	UserPic  string
}

type UserInfoUpdateReq struct {
	Id       int32
	UserName string
	NickName string
	Email    string
}

type UserAvatarUpdateReq struct {
	Id        int32
	AvatarUrl string
}
