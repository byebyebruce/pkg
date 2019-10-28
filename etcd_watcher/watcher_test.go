package etcd_watcher

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"
)

type listener struct {
}

func (*listener) Set(key []byte, val []byte) {
	fmt.Println("Set", string(key), string(val))
}
func (*listener) Create(key []byte, val []byte) {
	fmt.Println("Create", string(key), string(val))
}
func (*listener) Modify(key []byte, val []byte) {
	fmt.Println("Modify", string(key), string(val))
}
func (*listener) Delete(key []byte) {
	fmt.Println("Delete", string(key))
}

func Test_EtcdWatcher(t *testing.T) {
	ew, _ := NewEtcdWatcher([]string{"gate.sanguo.bj:2379"})
	key := fmt.Sprintf("%d/%d", time.Now().UnixNano(), 1)
	ew.cli.Put(context.Background(), key, "test")
	ew.AddWatch(key, true, &listener{})
	time.Sleep(time.Second)
	ew.cli.Put(context.Background(), key, "test1")
	time.Sleep(time.Second)
	ew.cli.Put(context.Background(), path.Join(key, "xxx"), "test1")
	time.Sleep(time.Second)
	ew.cli.Delete(context.Background(), key)
	time.Sleep(time.Second)
	ew.RemoveWatch(key)

	time.Sleep(time.Second)
	ew.ClearWatch()
	ew.Close(true)

}
