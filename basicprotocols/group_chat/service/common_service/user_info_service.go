package common_service

import (
	"fmt"
	"manindexer/basicprotocols/group_chat/api/respond"
	"manindexer/common"
)

const (
	ManCodeSuccess = 1
)

type ManResp struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type MetaIDUserInfo struct {
	Metaid  string `json:"metaid"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Avatar  string `json:"avatar"`
}

func FetchMetaIDUserInfo(address string) *respond.UserInfo {
	userInfo, err := fetchMetaIDUserInfoInfo(address)
	if err != nil {
		return nil
	}
	avatarImage := ""
	if userInfo.Avatar != "" {
		avatarImage = common.Config.GroupChat.ManHost + userInfo.Avatar
	}
	return &respond.UserInfo{
		Metaid:      userInfo.Metaid,
		AvatarImage: avatarImage,
		Avatar:      userInfo.Avatar,
		Name:        userInfo.Name,
	}
}

func fetchMetaIDUserInfoInfo(address string) (*MetaIDUserInfo, error) {
	var (
		url    string
		result string
		resp   *ManResp
		data   *MetaIDUserInfo
		err    error
	)
	query := map[string]string{}
	if common.Config.GroupChat.ManHost == "" {
		return nil, fmt.Errorf("manHost is empty")
	}
	url = fmt.Sprintf("%s/api/info/address/%s", common.Config.GroupChat.ManHost, address)

	result, err = common.GetUrl(url, query, nil)
	if err != nil {
		return nil, err
	}
	if err = common.JsonToObject(result, &resp); err != nil {
		return nil, fmt.Errorf("get request err:%s", err.Error())
	}
	if resp.Code != ManCodeSuccess {
		return nil, fmt.Errorf("msg:%s", resp.Message)
	}

	if err = common.JsonToAny(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("get request err:%s", err.Error())
	}
	return data, nil
}
