package tg_client

import (
	"context"
)

func (t *Client) HasTask(taskName string) bool {
	_, has := t.taskMap.Load(taskName)
	return has
}

func (t *Client) RunTask(taskName string, taskFun func(ctx context.Context, client *Client) error, onDone func(err error)) {
	taskCtx, cancelFun := context.WithCancel(t.ctx)
	_, has := t.taskMap.LoadOrStore(taskName, cancelFun)
	if has {
		return
	}
	go func() {
		defer func() {
			cancelFun()
			t.taskMap.Delete(taskName)
		}()
		doneChan := make(chan error)
		go func() {
			err := taskFun(taskCtx, t)
			doneChan <- err
		}()
		select {
		case err := <-doneChan:
			onDone(err)
		case <-taskCtx.Done():
			onDone(nil)
		}
	}()
}

func (t *Client) RemoveTask(taskName string) {
	cancelFun, has := t.taskMap.LoadAndDelete(taskName)
	if has {
		cancelFun.(context.CancelFunc)()
	}
}
