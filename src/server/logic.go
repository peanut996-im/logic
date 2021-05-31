// Package server
// @Title  logic.go
// @Description  封装业务逻辑
// @Author  peanut996
// @Update  peanut996  2021/5/31 10:43
package server

import (
	"fmt"
	"framework/api"
	"framework/api/model"
	"framework/logger"
	"framework/tool"
)

func (s *Server) PushChatMessage(message *model.ChatMessage) {
	ch := make(chan struct{}, 0)
	defer close(ch)
	pCR := &api.PushChatRequest{
		Message: message,
	}
	roomID := message.To
	room, err := model.GetRoomByID(roomID)
	if err != nil {
		logger.Error("Logic.PushChat no such room: %v", roomID)
		return
	}
	targets := []string{}
	if room.OneToOne {
		logger.Debug("Logic.Chat Push Friend Message")
		//single
		targets, err = model.GetFriendsByRoomID(room.RoomID)
		if err != nil {
			return
		}

	} else {
		//group
		targets, err = model.GetUserIDsByGroupID(message.To)
		if err != nil {
			logger.Error("Logic.PushChat Get Group Users err: %v", err)
			return
		}
	}
	for _, target := range targets {
		pCR.Target = target
		go s.SendToTarget(ch, target, pCR)
		<-ch
	}
}

func (s *Server) GetGateAddrFromScene(scene string) (string, error) {
	//TODO for cluster fix get from server
	return fmt.Sprintf("%v:%v", s.cfg.Gate.Host, s.cfg.Gate.Port), nil
}

func (s *Server) GetChatUrlFromScene(scene string) (string, error) {
	addr, err := s.GetGateAddrFromScene(scene)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%v/%v", addr, api.EventChat), nil
}

func (s *Server) SendToTarget(ch chan struct{}, target string, request *api.PushChatRequest) {
	logger.Debug("Logic.SendToTarget target: %v", target)
	url, err := s.GetChatUrlFromScene(target)
	if err != nil {
		logger.Error("Logic.PushChat GetGateUrl err: %v", err)
		return
	}
	go s.httpClient.GetGoReq().Post(url).Send(request).End()
	ch <- struct{}{}
}

func (s *Server) InviteFriendsToGroup(friends []string, groupID string) error {
	for _, friend := range friends {
		err := model.CreateGroupUser(groupID, friend)
		if err != nil {
			return err
		}
	}
	// TODO 服务端主动推送客户端刷新
	return nil
}

func (s *Server) PullMessageByPage(uid, friendID, groupID string, current, pageSize int64) ([]*model.ChatMessage, error) {
	if len(friendID) > 0 {
		friend, err := model.GetFriend(uid, friendID)
		if err != nil {
			return nil, err
		}
		messages, err := model.GetFriendMessageWithPage(friend, current, pageSize)
		if err != nil {
			return nil, err
		}
		return messages, nil
	} else if len(groupID) > 0 {
		group := &model.Group{GroupID: groupID}
		messages, err := model.GetGroupMessageWithPage(group, current, pageSize)
		if err != nil {
			return nil, err
		}
		return messages, nil
	}
	return nil, api.ErrorCodeToError(api.ErrorHttpParamInvalid)
}

func (s *Server) UpdateUserInfo(uid string, account string, password string, avatar string) (*model.User, error) {
	user, err := model.GetUserByUID(uid)
	if nil != err {
		return nil, err
	}

	if len(account) > 0 {
		user.Account = account
	} else if len(password) > 0 {
		cipher := tool.EncryptBySha1(fmt.Sprintf("%v%v", password, s.cfg.AppKey))
		user.Password = cipher
	} else if len(avatar) > 0 {
		user.Avatar = avatar
	}
	err = model.UpdateUser(user)
	if nil != err {
		return nil, err
	}
	u, err := model.GetUserByUID(user.UID)
	if nil != err {
		return nil, err
	}
	return u, nil
}
