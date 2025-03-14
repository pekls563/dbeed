package biz

import (
	"bigEventProject/account_srv/account_srv_params"
	"bigEventProject/account_srv/internal"
	"bigEventProject/account_srv/model"
	"crypto/md5"
	"errors"
	"github.com/anaskhan96/go-password-encoder"
)

type AccountServer struct {
}

func (a AccountServer) UserRegister(req *account_srv_params.UserRegisterReq, res *account_srv_params.Empty) error {

	var account model.Account
	result := internal.DB.Where(&model.Account{UserName: req.UserName}).First(&account)
	if result.RowsAffected == 1 {
		return errors.New("账户已存在")
	}
	account.UserName = req.UserName

	//密码使用md5加密
	options := password.Options{
		SaltLen:      16,
		Iterations:   100,
		KeyLen:       32,
		HashFunction: md5.New,
	}

	//根据配置进行md5加密并返回盐值和加密后的密码
	salt, encodePwd := password.Encode(req.Password, &options)

	account.Salt = salt
	account.Password = encodePwd

	r := internal.DB.Create(&account)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}
	return nil
}

func (a AccountServer) UserLogin(req *account_srv_params.UserLoginReq, res *account_srv_params.UserLoginRes) error {

	var account model.Account
	result := internal.DB.Where(&model.Account{UserName: req.UserName}).First(&account)
	if result.RowsAffected < 1 {
		return errors.New("账户不存在")
	}
	options := password.Options{
		SaltLen:      16,
		Iterations:   100,
		KeyLen:       32,
		HashFunction: md5.New,
	}
	r := password.Verify(req.Password, account.Salt, account.Password, &options)
	if r == false {
		return errors.New("密码错误")
	}
	//var userLoginRes pb_account.UserLoginRes
	//userLoginRes.Id = int32(account.ID)
	(*res).Id = int32(account.ID)

	return nil

}

func (a AccountServer) UserInfo(req *account_srv_params.UserInfoReq, res *account_srv_params.UserInfoRes) error {

	var account model.Account
	result := internal.DB.First(&account, req.Id)
	if result.RowsAffected == 0 {
		return errors.New("账户不存在")
	}

	(*res).Id = int32(account.ID)
	(*res).UserName = account.UserName
	(*res).Email = account.Email
	(*res).UserPic = account.AvatarUrl
	(*res).NickName = account.NickName
	return nil

}

func (a AccountServer) UserInfoUpdate(req *account_srv_params.UserInfoUpdateReq, res *account_srv_params.Empty) error {

	var account model.Account
	result := internal.DB.First(&account, req.Id)
	if result.RowsAffected == 0 {
		return errors.New("账户不存在")
	}

	account.UserName = req.UserName
	account.Email = req.Email
	account.NickName = req.NickName
	r := internal.DB.Save(&account)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}
	return nil
}

func (a AccountServer) UserAvatarUpdate(req *account_srv_params.UserAvatarUpdateReq, res *account_srv_params.Empty) error {

	var account model.Account
	result := internal.DB.First(&account, req.Id)
	if result.RowsAffected == 0 {
		return errors.New("账户不存在")
	}
	account.AvatarUrl = req.AvatarUrl
	r := internal.DB.Save(&account)
	if r.Error != nil {
		return errors.New("服务端内部错误")
	}
	return nil

}
