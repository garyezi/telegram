package tg_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/tg"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var AuthError = errors.New("this account login has expired")

func (t *Client) Self(ctx context.Context) (*tg.User, error) {
	status, err := t.client.Auth().Status(ctx)
	if err != nil {
		return nil, err
	}
	if !status.Authorized {
		return nil, AuthError
	} else {
		return status.User, nil
	}
}

func (t *Client) CheckCanSendMessage(ctx context.Context) (canSend bool, utilTime *time.Time, err error) {
	peer, err := t.Sender().Resolve("@SpamBot").AsInputPeer(ctx)
	if err != nil {
		return false, nil, err
	}
	_, err = t.Sender().PeerPromise(func(ctx context.Context) (tg.InputPeerClass, error) {
		return peer, nil
	}).Text(context.Background(), "/start")
	if err != nil {
		return false, nil, err
	}
	time.Sleep(time.Second * 3)
	history, err := t.API().MessagesGetHistory(context.Background(), &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 1,
	})
	if err != nil {
		return false, nil, err
	}
	messages := history.(*tg.MessagesMessagesSlice)
	msg := messages.Messages[0].(*tg.Message)
	switch true {
	case strings.Contains(msg.Message, "I’m afraid some Telegram"):
		// 定义正则表达式匹配时间
		re := regexp.MustCompile(`\d{1,2} [A-Za-z]{3} \d{4}, \d{2}:\d{2} UTC`)
		match := re.FindString(msg.Message)
		if match != "" {
			// 定义时间格式
			const layout = "2 Jan 2006, 15:04 MST" // Go 的时间格式必须是固定参考值
			parsedTime, err := time.Parse(layout, match)
			if err != nil {
				fmt.Println("时间解析失败:", err)
				return false, nil, err
			} else {
				fmt.Println("提取并转化的时间:", parsedTime)
				return false, &parsedTime, nil
			}
		} else {
			fmt.Println("未找到时间")
		}
		return false, nil, nil
	case strings.Contains(msg.Message, "I’m very sorry that you had to contact me"):
		return false, nil, nil
	default:
		return true, nil, nil
	}
}

func (t *Client) InitEntity(ctx context.Context) {
	iter := query.GetDialogs(t.client.API()).BatchSize(100).Iter()
	for iter.Next(ctx) {
		result := iter.Value()
		switch v := result.Dialog.GetPeer().(type) {
		default:
			marshal, err := json.Marshal(v)
			if err != nil {
				return
			}
			fmt.Println(string(marshal))
			break
		}
	}
	fmt.Println(iter.Err())
}

func (t *Client) GetEntity(ctx context.Context, target string) (interface{}, error) {
	if _, err := strconv.Atoi(target); err == nil {
		return nil, errors.New("暂不支持ID进行查询实体")
	} else {
		username, err := t.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: target,
		})
		if err != nil {
			return nil, err
		}
		//return username, nil
		if len(username.Chats) > 0 {
			return username.Chats[0], nil
		} else if len(username.Users) > 0 {
			return username.Users[0], nil
		} else {
			return nil, errors.New("未找到实体")
		}
	}
}

func (t *Client) ApplyBoost(ctx context.Context, username string) error {
	boosts, err := t.API().PremiumGetMyBoosts(ctx)
	if err != nil {
		return err
	}
	var slotsId int
	for _, boost := range boosts.MyBoosts {
		if _, ok := boost.GetCooldownUntilDate(); !ok {
			slotsId = boost.Slot
			break
		}
	}
	if slotsId == 0 {
		return errors.New("没有可用插槽")
	}
	result, err := t.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return err
	}
	var inputPeer tg.InputPeerClass
	switch v := result.Chats[0].(type) {
	case *tg.Channel:
		inputPeer = &tg.InputPeerChannel{
			ChannelID:  v.ID,
			AccessHash: v.AccessHash,
		}
		break
	default:
		return errors.New("非Channel类型")
	}
	_, err = t.API().PremiumApplyBoost(ctx, &tg.PremiumApplyBoostRequest{
		Slots: []int{slotsId},
		Peer:  inputPeer,
	})
	return err
}
