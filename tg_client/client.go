package tg_client

import (
	"context"
	"errors"
	"fmt"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type Client struct {
	// 客户端配置
	config *Config
	// 客户端上下文
	ctx context.Context
	//终止的回调
	onDone func(err error)
	//终止客户端方法
	cancelFun context.CancelFunc
	// 客户端本体
	client *telegram.Client
	// 事件处理
	dispatcher tg.UpdateDispatcher
	// 洪水中间件
	waiter *floodwait.Waiter
	// 消息发送器
	sender *message.Sender
	// 任务Map
	taskMap sync.Map
}

type Config struct {
	Session session.Storage
	Proxy   *Proxy
	OnDone  func(err error)
}

func NewClient(config *Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	account := &Client{
		config:    config,
		ctx:       ctx,
		cancelFun: cancel,
		onDone:    config.OnDone,
	}
	account.dispatcher = tg.NewUpdateDispatcher()
	account.waiter = floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		fmt.Println("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})
	clientOption := telegram.Options{
		SessionStorage: account.config.Session,
		UpdateHandler:  account.dispatcher,
		Middlewares: []telegram.Middleware{
			account.waiter,
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
	}
	if config.Proxy != nil {
		dc := config.Proxy.GetDial()
		clientOption.Resolver = dcs.Plain(dcs.PlainOptions{
			Dial: dc.DialContext,
		})
	}
	account.client = telegram.NewClient(2040, "b18441a1ff607e10a989891a5462e627", clientOption)
	account.sender = message.NewSender(account.client.API())
	return account
}

func (t *Client) Config() tg.Config {
	return t.client.Config()
}

func (t *Client) API() *tg.Client {
	return t.client.API()
}

func (t *Client) Sender() *message.Sender {
	return t.sender
}

func (t *Client) Dispatcher() *tg.UpdateDispatcher {
	return &t.dispatcher
}

func (t *Client) Done(err error) {
	if t.onDone != nil {
		t.onDone(err)
	}
	t.cancelFun()
}

func (t *Client) Start() error {
	errChan := make(chan error, 1)
	initDone := make(chan struct{})
	go func() {
		defer close(errChan)
		errChan <- t.waiter.Run(t.ctx, func(ctx context.Context) error {
			return t.client.Run(ctx, func(ctx context.Context) error {
				close(initDone)
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						_, err := t.Self(ctx)
						if err != nil && errors.Is(err, AuthError) {
							return err
						}
						time.Sleep(time.Second * 5)
					}
				}
			})
		})
	}()
	select {
	case <-t.ctx.Done():
		t.Done(t.ctx.Err())
		return t.ctx.Err()
	case err := <-errChan:
		t.Done(err)
		return err
	case <-initDone:
		return nil
	}
}
