package internal

import (
	"log"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

type ZK struct {
	Conn    *zk.Conn
	servers []string
	ch      <-chan zk.Event
	watcher func(event zk.Event)
}

func NewZK(servers []string, watcher func(zk.Event)) *ZK {
	var zookeeper = &ZK{}
	var wg = &sync.WaitGroup{}

	zookeeper.servers = servers
	zookeeper.watcher = watcher

	wg.Add(1) //Add用来设置等待线程数量

	go func() {
		var ech <-chan zk.Event
		var err error
		var c *zk.Conn
		if zookeeper.Conn == nil {
			c, ech, err = zk.Connect(servers, time.Second) //*10)
			zookeeper.Conn = c
		}

		if err != nil {
			log.Printf("zk Connect ::: " + err.Error())
			panic(err)
		}

		go func() {
			for {
				select {
				case ch := <-ech:
					{
						switch ch.State {
						case zk.StateConnecting:
							{
								log.Println("StateConnecting")
							}
						case zk.StateConnected: //若链接后出现断连现象 重连时会报异常 因为此时同步信号量已为0
							{
								wg.Done()
								log.Println("StateConnected")
							}
						case zk.StateExpired:
							{
								log.Println("StateExpired")
							}
						case zk.StateDisconnected:
							{
								wg.Add(1)
								log.Println("StateDisconnected")
							}
						}

						DoWatch(ch, watcher)

					}
				}
			}
		}()

	}()

	wg.Wait()
	return zookeeper
}

func DoWatch(zkEvent zk.Event, watcher func(zk.Event)) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}

		//log.Println("test DoWatch!!!")
	}()

	watcher(zkEvent)
}

func (zookeeper *ZK) Create(path string, data []byte, version int32) error {

	err := zookeeper.do(func(conn *zk.Conn) error {
		_, err := conn.Create(path, data, 0, zk.WorldACL(zk.PermAll))
		return err
	})

	return err
}

func (zookeeper *ZK) Set(path string, data []byte, version int32) error {

	err := zookeeper.do(func(conn *zk.Conn) error {
		_, err := conn.Set(path, data, version)
		return err
	})

	return err
}

func (zookeeper *ZK) Delete(path string, version int32) error {

	err := zookeeper.do(func(conn *zk.Conn) error {

		err := conn.Delete(path, version)
		return err
	})

	return err
}

func (zookeeper *ZK) Get(path string) (string, error) {
	var res []byte
	err := zookeeper.do(func(conn *zk.Conn) error {
		r, _, err := conn.Get(path)
		res = r
		return err
	})

	return string(res), err
}

func (zookeeper *ZK) GetW(path string) (string, <-chan zk.Event, error) {
	var res []byte
	var c <-chan zk.Event
	err := zookeeper.do(func(conn *zk.Conn) error {
		var err error
		res, _, c, err = conn.GetW(path)
		return err
	})

	return string(res), c, err
}

func (zookeeper *ZK) GetChildren(path string) ([]string, error) {

	var res []string

	err := zookeeper.do(func(conn *zk.Conn) error {
		r, _, err := conn.Children(path)
		res = r
		return err
	})

	return res, err
}

func (zookeeper *ZK) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {

	var res []string
	var c <-chan zk.Event
	err := zookeeper.do(func(conn *zk.Conn) error {
		var err error
		res, _, c, err = conn.ChildrenW(path)

		if err != nil {
			log.Println("GetChildrenW ::::" + err.Error())
		}
		return err
	})

	return res, c, err
}

func (zookeeper *ZK) do(fn func(conn *zk.Conn) error) error {
	conn := zookeeper.Conn
	//conn != nil && conn.State() == zk.StateConnected
	if conn != nil {
		err := fn(conn)
		return err
	}

	//现有逻辑很难执行以下操作
	log.Println("renew zk")
	oldOne := zookeeper

	newOne := NewZK(zookeeper.servers, zookeeper.watcher)

	oldOne.Conn.Close()

	err := fn(newOne.Conn)

	return err
}

func (zookeeper *ZK) Exists(path string) (bool, error) {

	var exist bool

	err := zookeeper.do(func(conn *zk.Conn) error {
		b, _, err := conn.Exists(path)
		exist = b
		return err
	})

	return exist, err
}
