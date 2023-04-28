package envoy

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cicd-core/util"
)

type ListenerProtocol string

const (
	LpTcp   ListenerProtocol = "TCP"
	LpHttp  ListenerProtocol = "HTTP"
	LpHttp2 ListenerProtocol = "HTTP2"
)

type RouteMatchType int

const (
	Path   RouteMatchType = 0
	Prefix RouteMatchType = 1
	Regex  RouteMatchType = 2
)

type UnitType int

const (
	Unit_Second UnitType = 0 //秒
	Unit_Minute UnitType = 1 //分钟
	Unit_Hour   UnitType = 2 //小时
	Unit_Day    UnitType = 3 //天
)

type LimitType int

const (
	Limit_Route    LimitType = 0 //路由限速
	Limit_Cluster  LimitType = 1 //服务限速
	Limit_ClientIp LimitType = 2 //客户端Ip限速
	Limit_Guid     LimitType = 3 //Guid限速
)

type CityPrefix int

const (
	CityPrefixEnable  CityPrefix = 0
	CityPrefixDisable CityPrefix = 1
)

const (
	LOCAL_REDIS_CHECK = 0
	AUTH_SERVER_CHECK = 1
)

type K8sKind int

const (
	K8sKind_Image      K8sKind = -1 //镜像
	K8sKind_Expanded   K8sKind = 1  //可扩展部署 Sidecar
	K8sKind_TimeJob    K8sKind = 2  //定时任务
	K8sKind_JobWithEnv K8sKind = 3  //带有特定环境变量的Job
	K8sKind_Fronted    K8sKind = 4  //前端
	K8sKind_Job        K8sKind = 5  //JOB
	K8sKind_Gateway    K8sKind = 6  //网关
)

func (lp ListenerProtocol) ToString() string {
	return fmt.Sprint(lp)
}

type LDS struct {
	Name           string
	RouteMatchType RouteMatchType
	Version        string
	Plugins        []*Plugins
	Listeners      []*Listener
	EnableMirror   bool
}

func (l *LDS) WithPlugin(pluginName string) bool {
	for _, v := range l.Plugins {
		if v.PluginName == pluginName {
			return true
		}
	}

	return false
}

type Plugins struct {
	PluginName      string
	FunctionalTypes int32
}

type Listener struct {
	Routes    []*HTTPRoute
	Domains   []string
	EnableTLS bool
}

var Cities = []string{"sz", "sh", "wx", "cz", "ks"}

type HTTPRoute struct {
	Prefix                string
	PrefixRewrite         string
	HostRewrite           bool
	ClusterName           string //对应k8s中的service name
	Auth                  int
	GuId                  string
	Verify                string
	Version               string
	AbTest                *AbTest
	RateLimits            *RateLimit
	TimeOut               int
	VerifyParam           bool
	HashRoute             bool
	HashRouteParams       []string
	BindInternalUsers     bool
	BindInternalUserTag   string
	IsEnableAB            bool       //是否启用AB服务
	ABTags                []*ABTag   //AB路由中的服务标签
	IsEnableCanary        bool       //是否开启了灰度
	CanaryTag             *ABTag     //灰度中的服务标签
	Method                string     //OpenApi路由参数
	IfNeedVerifySign      bool       //OpenApi是否需要验证签名
	IfNeedEncryptContent  bool       //OpenApi是否需要加密发送数据
	AuthServerVersion     int        //是否采用v2版本的认证 // 0 代表redis认证 1 代表 v2版本走auth server认证
	IsCityTagPrefixEnable CityPrefix //是否带城市路由 /sz /ks /sh /wx  0 表示带城市表示 1 表示不带城市表示
}

type ABTag struct {
	Sceance     string `json:"sceans"`     //AB场景的名称,比如search(搜索), category(分类)等
	SceanceName string `json:"sceansName"` //AB场景中的分组名称, 比如A, B等
	Tag         string `json:"tag"`        //AB场景中对应的服务标签 比如 prod, canary等
}

type AbTest struct {
	IsEnable bool
	Mark     string
	Policy   int
}

type RateLimit struct {
	IsEnable     bool
	LimitActions []*LimitAction
}

type LimitAction struct {
	IsEnable  bool      //是否启用
	Type      LimitType //限速类型
	Threshold int       //阈值
	Unit      UnitType  //单位类型
}

type Cluster struct {
	Name      string
	EDS       *EDS
	HashRoute bool
}

type EDS struct {
	Endpoints        []*Endpoint
	Name             string
	Version          string
	ClusterIp        string
	Ports            []*Ports
	NetflowTagEnable int // -1 禁用  0 启用
	EDSVersions      []EDS_Version
	CircuitBreaker   []*CircuitBreaker
	OutlierDetection OutlierDetection
	K8SKindId        EDS_K8SKindId

	/**
	是否开启 熔断 1-启用,0-禁用,参照枚举 TrueOrFalse
	*/
	EnableCircuitbreaker int

	/**
	是否开启 异常点检测 1-启用,0-禁用,参照枚举 TrueOrFalse
	*/
	EnableOutlierDetection util.TrueOrFalse

	/**
	是否开启 健康检查 1-启用,0-禁用,参照枚举 TrueOrFalse
	*/
	EnableHealthCheck util.TrueOrFalse

	/**
	健康检查
	*/
	HealthCheck HealthCheck

	GrayStrategy int
}

type HealthCheck struct {

	/**
	每次健康检查扫描的间隔时间
	*/
	Interval int64
	/**
	http健康检查
	*/
	HttpHealthCheck HttpHealthCheck
}

type HttpHealthCheck struct {
	/**
	路径
	*/
	Path string
}

type EDS_K8SKindId int

const (
	Nil         EDS_K8SKindId = iota
	ExtenDeploy               = 1
	CronJob                   = 2
	EnvJob                    = 3
	FrontDeploy               = 4
	Job                       = 5
	Image                     = 0
)

const (
	NONE    = 0
	BETA    = 2
	SIDECAR = 3
)

type Ports struct {
	Name       string
	Protocol   string
	Port       int
	NodePort   int
	TargetPort int
}

func (ports *Ports) GetProtocol() ListenerProtocol {
	protocols := strings.Split(ports.Name, "-")

	if len(protocols) < 2 {
		return LpTcp
	}

	var protocol ListenerProtocol

	switch protocols[1] {
	case "tcp":
		protocol = LpTcp
		break
	case "http":
		protocol = LpHttp
		break
	case "http2":
		protocol = LpHttp2
		break
	default:
		protocol = LpTcp
		break
	}

	return protocol
}

type EDS_Version_Policy int

const (
	A EDS_Version_Policy = iota
	B                    = 1
	C                    = 2
)

type Flow_Control_Policy int

const (
	Policy_Gray   Flow_Control_Policy = iota
	Policy_Weight                     = 1
)

type EDS_Version struct {
	Version      string
	Enable       util.YesOrNo
	SelectPolicy Flow_Control_Policy
	FlowWeight   int
	Policys      []EDS_Version_Policy
	NetflowTag   []string
}

type CircuitBreaker struct {
	/**
	是否启用熔断
	*/

	Priority           int32
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
	TrackRemaining     bool
	MaxConnectionPools uint32
}

/**
异常点检测
*/
type OutlierDetection struct {
	Consecutive_5Xx          uint32
	Interval                 uint32
	BaseEjectionTime         uint32
	MaxEjectionPercent       uint32
	EnforcingConsecutive_5Xx uint32
}

type TimeOutPolicy struct {
	IdleTimeOut int
}

func (eds *EDS) GetEndpoint() (endpoint *Endpoint, error error) {
	var ep, err = doroundrobin(eds)

	if err != nil {
		return nil, err
	}

	return ep, nil
}

func (eds *EDS) GetVersions() []string {
	var eps []string = make([]string, 0)
	if eds.Endpoints == nil {
		return eps
	}

	epKV := make(map[string]string)

	for _, v := range eds.Endpoints {
		if v.Version == "" {
			continue
		}

		if epKV[v.Version] == "" {
			epKV[v.Version] = v.Version
			eps = append(eps, v.Version)
		}
	}

	return eps
}

func (eds *EDS) GetPolicys() []EDS_Version_Policy {
	var evps = make([]EDS_Version_Policy, 0)
	if eds.EDSVersions == nil {
		return evps
	}

	epKV := make(map[EDS_Version_Policy]*EDS_Version_Policy)

	for _, v := range eds.EDSVersions {
		if v.Policys == nil {
			continue
		}

		for _, p := range v.Policys {
			if epKV[p] == nil {
				epKV[p] = &p
				evps = append(evps, p)
			}
		}
	}

	return evps
}

func (eds *EDS) GetCdsVersion() string {
	if eds == nil || eds.Endpoints == nil || len(eds.Endpoints) == 0 {
		return ""
	}

	// 获取endpoint名字数组
	var podNames []string
	for _, ep := range eds.Endpoints {
		podNames = append(podNames, ep.Name)
	}

	// 获取EDSVersions版本名字以及禁用启用状态
	var edsVersions []string
	var enable string
	for _, edsVersion := range eds.EDSVersions {
		if edsVersion.Enable == util.No {
			enable = "false"
		} else {
			enable = "true"
		}
		edsVersions = append(edsVersions, edsVersion.Version+"-"+enable)
	}

	// 获取endpoint IP数组
	var endPointsVersion []string
	for _, endpoint := range eds.Endpoints {
		endPointsVersion = append(endPointsVersion, endpoint.Ip)
	}

	r := strings.Join(podNames, "*") + strings.Join(edsVersions, "*") + strings.Join(endPointsVersion, "*") + eds.ClusterIp

	byte16 := md5.Sum([]byte(r))

	return fmt.Sprintf("%x", byte16)
}

var curIndex = 0

func doroundrobin(des *EDS) (inst *Endpoint, err error) {

	if len(des.Endpoints) == 0 {
		err = errors.New("no endpoints")
		return
	}
	lens := len(des.Endpoints)
	if curIndex >= lens {
		curIndex = 0
	}

	inst = des.Endpoints[curIndex]
	curIndex = (curIndex + 1) % lens
	return
}

type Endpoint struct {
	Namespace string
	Ip        string
	Port      int
	Weight    int
	Protocol  string
	Version   string
	Name      string

	/**
	Yes-启用,No-禁用
	*/
	Status util.YesOrNo
}

func (ed *Endpoint) ToString() string {
	if ed == nil {
		return ""
	}

	return stringJoin("http://", ed.Ip, ":", strconv.FormatInt(int64(ed.Port), 10))
}

func stringJoin(args ...string) string {

	if len(args) == 0 {
		return ""
	}
	var buffer bytes.Buffer

	for _, arg := range args {
		buffer.WriteString(arg)
	}

	return buffer.String()
}
