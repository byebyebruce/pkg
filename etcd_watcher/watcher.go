package etcd_watcher

import (
	"context"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var (
	timeOut = time.Duration(3) * time.Second // 超时市场
)

// Listener
type Listener interface {
	Set([]byte, []byte)
	Create([]byte, []byte)
	Modify([]byte, []byte)
	Delete([]byte)
}

// EtcdWatcher ETCD key监视器
type EtcdWatcher struct {
	cli          *clientv3.Client
	wg           sync.WaitGroup
	listener     Listener
	mu           sync.Mutex
	closeHandler map[string]func()
}

// NewEtcdWatcher 构造
func NewEtcdWatcher(servers []string) (*EtcdWatcher, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   servers,
		DialTimeout: timeOut,
	})
	if err != nil {
		return nil, err
	}

	em := &EtcdWatcher{
		cli:          cli,
		closeHandler: make(map[string]func()),
	}

	return em, nil
}

// AddWatch 添加监视
func (mgr *EtcdWatcher) AddWatch(key string, prefix bool, target Listener) bool {
	mgr.mu.Lock()
	if _, ok := mgr.closeHandler[key]; ok {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	mgr.closeHandler[key] = cancel
	mgr.mu.Unlock()

	mgr.wg.Add(1)
	go mgr.watch(ctx, key, prefix, target)

	return true
}

// RemoveWatch 删除监视
func (mgr *EtcdWatcher) RemoveWatch(key string) bool {
	mgr.mu.Lock()
	cancel, ok := mgr.closeHandler[key]
	if !ok {
		return false
	}
	cancel()
	delete(mgr.closeHandler, key)
	mgr.mu.Unlock()

	return true
}

// ClearWatch 清除所有监视
func (mgr *EtcdWatcher) ClearWatch() {

	mgr.mu.Lock()
	for k := range mgr.closeHandler {
		mgr.closeHandler[k]()
	}
	mgr.closeHandler = make(map[string]func())
	mgr.mu.Unlock()

}

// Close 关闭
func (mgr *EtcdWatcher) Close(wait bool) {
	mgr.ClearWatch()

	if wait {
		mgr.wg.Wait()
	}

	mgr.cli.Close()
	mgr.cli = nil
}

func (mgr *EtcdWatcher) watch(ctx context.Context, key string, prefix bool, listener Listener) error {
	defer mgr.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	var resp *clientv3.GetResponse
	var err error
	if prefix {
		resp, err = mgr.cli.Get(ctx, key, clientv3.WithPrefix())
	} else {
		resp, err = mgr.cli.Get(ctx, key)
	}
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		listener.Set(ev.Key, ev.Value)
	}

	var watchChan clientv3.WatchChan
	if prefix {
		watchChan = mgr.cli.Watch(context.Background(), key, clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision+1))
	} else {
		watchChan = mgr.cli.Watch(context.Background(), key, clientv3.WithRev(resp.Header.Revision+1))
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case resp := <-watchChan:
			err := wresp.Err()
			if err != nil {
				return err
			}
			for _, ev := range resp.Events {
				if ev.IsCreate() {
					listener.Create(ev.Kv.Key, ev.Kv.Value)
				} else if ev.IsModify() {
					listener.Modify(ev.Kv.Key, ev.Kv.Value)
				} else if ev.Type == mvccpb.DELETE {
					listener.Delete(ev.Kv.Key)
				} else {
				}
			}
		}
	}
}
