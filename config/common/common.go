package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"cicd-core/config/envoy"
	"cicd-core/util"
	zk2 "cicd-core/zookeeper"

	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

var Paths = "config,lds,cds,connection,service"

type ChangedEvent struct {
	Path string
}

type Manager struct {
	configMap                 sync.Map
	CdsChangedEvent           chan ChangedEvent
	LdsChangedEvent           chan ChangedEvent
	InternalUsersChangedEvent chan ChangedEvent
}

var manager *Manager
var once sync.Once
var zoo *zk2.ZK

func NewManager() *Manager {
	once.Do(func() {
		manager = &Manager{}
		manager.CdsChangedEvent = make(chan ChangedEvent)
		manager.LdsChangedEvent = make(chan ChangedEvent)
		manager.InternalUsersChangedEvent = make(chan ChangedEvent)
		zks := []string{"zk01:2181", "zk02:2181", "zk03:2181"}
		zoo = zk2.NewZK(zks, func(event zk.Event) {
			switch event.State {
			case zk.StateExpired:
				{
					log.Println("StateExpired")
					setAll()
					log.Println("setAll /")
				}
			}
			switch event.Type {
			case zk.EventNodeDataChanged:
				{
					manager.update(event.Path)
					log.Printf("EventNodeDataChanged: %s\n", event.Path)
					if strings.HasPrefix(event.Path, "/cds/") {
						go func() {
							defer func() {
								if err := recover(); err != nil {
									log.Println(err)
								}
							}()

							manager.CdsChangedEvent <- ChangedEvent{
								Path: event.Path,
							}
						}()
					}
					if strings.HasPrefix(event.Path, "/lds/") {
						go func() {
							defer func() {
								if err := recover(); err != nil {
									log.Println(err)
								}
							}()

							manager.LdsChangedEvent <- ChangedEvent{
								Path: event.Path,
							}
						}()
					}
				}
			case zk.EventNodeDeleted:
				{
					manager.delete(event.Path)
					log.Printf("EventNodeDeleted: %s\n", event.Path)
				}
			case zk.EventNodeChildrenChanged:
				{
					log.Printf("EventNodeChildrenChanged: %s\n", event.Path)
				}
			case zk.EventNodeCreated:
				{
					log.Printf("EventNodeCreated: %s\n", event.Path)
				}
			}
		})

		setAll()
		go func() {
			for {
				select {
				case <-time.After(30 * time.Second):
					{

						if cbs != nil {
							for _, cb := range cbs {
								err := cb(zoo)
								if err != nil {
									log.Println(err)
								}
							}
						}

					}
				}
			}
		}()
	})

	return manager
}

var mutex sync.Mutex

var cbs = make([]func(zk *zk2.ZK) error, 0)

func (m *Manager) Register(callback func(zk *zk2.ZK) error) {
	mutex.Lock()
	defer mutex.Unlock()

	cbs = append(cbs, callback)
}

func convert2String(o interface{}) (string, error) {
	value, ok := o.(string)
	if !ok {
		log.Println("")

		return "", errors.New("convet to string failed")
	}
	return value, nil
}

func (m *Manager) Get(configPath string) interface{} {

	if strings.HasPrefix(configPath, "/connection") || strings.HasPrefix(configPath, "/service") || strings.HasPrefix(configPath, "/config") || strings.HasPrefix(configPath, "/data") || strings.HasPrefix(configPath, "/cds") || strings.HasPrefix(configPath, "/lds") {
		_, ok := m.configMap.Load(configPath)

		if !ok {
			vv, _, err := zoo.GetW(configPath)

			if err == nil {
				m.set(configPath, vv)
				return m.Get(configPath)
			}
			return ""
		}
	}
	v, _ := m.configMap.Load(configPath)
	return v
}

/**
一次性获取zk所有数据
*/
func GetZkCdsData() map[string]*envoy.EDS {

	NewManager()

	mapResult := make(map[string]*envoy.EDS)

	path := "/cds"

	var zk = zoo

	//不再监听根目录变化，只监听节点变化
	children, err := zk.GetChildren(path)

	if err != nil {
		log.Println(fmt.Sprintf("path:%s :err:%s", path, err.Error()))
		panic(err)
	}

	if children == nil || len(children) <= 0 {
		return mapResult
	}

	for _, sPath := range children {

		value := new(envoy.EDS)
		var nextPath = path

		if strings.EqualFold(path, "/") && !(strings.Index(Paths, sPath) >= 0) {
			continue
		}

		if !strings.HasSuffix(nextPath, "/") {
			nextPath += "/"
		}
		nextPath += sPath
		v, _, _ := zk.GetW(nextPath)
		util.ByteToStruct([]byte(v), value)
		mapResult[sPath] = value
	}
	return mapResult
}

func initMap(path string) {

	if path == "" {
		return
	}

	if strings.EqualFold(path, "/") && !(strings.Index(Paths, path) >= 0) {
		return
	}

	var zk = zoo

	//不再监听根目录变化，只监听节点变化
	children, err := zk.GetChildren(path)

	if err != nil {
		log.Println(fmt.Sprintf("path:%s :err:%s", path, err.Error()))
		panic(err)
	}

	if children == nil || len(children) <= 0 {
		return
	}

	for _, sPath := range children {
		var nextPath = path

		if strings.EqualFold(path, "/") && !(strings.Index(Paths, sPath) >= 0) {
			continue
		}

		if !strings.HasSuffix(nextPath, "/") {
			nextPath += "/"
		}
		nextPath += sPath

		v, _, _ := zk.GetW(nextPath)

		manager.set(nextPath, v)
	}
}

func interface2ByteArray(v interface{}) ([]byte, error) {
	var data []byte
	var err error
	if vv, ok := v.(string); ok {
		data = []byte(vv)
	} else {
		data, err = json.Marshal(v)

		if err != nil {
			return nil, err
		}
	}

	return data, err
}

func (m *Manager) Create(name string, v interface{}) error {
	var zk = zoo

	data, err := interface2ByteArray(v)

	if err != nil {
		return err
	}

	err = zk.Create(name, data, -1)

	return err
}

func (m *Manager) Delete(name string) error {
	var zk = zoo

	err := zk.Delete(name, -1)
	if err != nil {
		manager.delete(name)
	}
	return err
}

func (m *Manager) Set(path string, v interface{}) error {
	var zk = zoo

	data, err := interface2ByteArray(v)

	if err != nil {
		return err
	}

	err = zk.Set(path, data, -1)

	return err
}

func (m *Manager) Exists(path string) bool {
	var zk = zoo

	b, _ := zk.Exists(path)

	return b
}

func (m *Manager) GetChildren(path string) []string {
	var zk = zoo

	children, _ := zk.GetChildren(path)

	return children
}

func (m *Manager) update(path string) {

	if path == "" {
		return
	}

	if strings.EqualFold(path, "/") && !(strings.Index(Paths, path) >= 0) {
		return
	}

	var zk = zoo

	v, _, err := zk.GetW(path)

	if err != nil {
		panic(err)
	}

	manager.set(path, v)
}

func (m *Manager) delete(path string) {

	if path == "" {
		return
	}

	m.configMap.Delete(path)
}

func setAll() {
	paths := strings.Split(Paths, ",")
	for _, v := range paths {
		initMap("/" + v)
	}
}

func (m *Manager) GetAll(configPath string) map[string]string {

	var mapping = make(map[string]string)

	m.configMap.Range(func(key, value interface{}) bool {

		if strings.HasPrefix(key.(string), configPath) {

			if vv, ok := value.(string); ok {
				mapping[key.(string)] = vv
			} else {
				data, err := json.Marshal(value)
				if err == nil {
					mapping[key.(string)] = string(data)
				} else {
					log.Println("GetAll", key, err)
				}
			}

		}

		return true
	})

	return mapping
}

func (m *Manager) Dispose() {
	zoo.Conn.Close()
}

func (m *Manager) set(k string, v string) {
	var value interface{}

	if strings.HasPrefix(k, "/lds") {
		value = &envoy.LDS{}
		_ = util.ByteToStruct([]byte(v), value)
	} else if strings.HasPrefix(k, "/cds") {
		value = &envoy.EDS{}
		_ = util.ByteToStruct([]byte(v), value)
	} else {
		value = v
	}

	m.configMap.Store(k, value)
}

func (m *Manager) SetCache(k string, v string) {
	m.set(k, v)
}
