package easy_discovery_consul

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/zhuxiujia/easyrpc_discovery"
	"strconv"
	"strings"
	"time"
)

type ConsulCheckType int

const (
	TCP ConsulCheckType = iota
	HTTP
)

type ConsulManager struct {
	ConsulAddress string
	reg           *consulapi.AgentServiceRegistration
	client        *consulapi.Client
	manager       *easyrpc_discovery.RpcServiceManager
	clearFunc     *func(m map[string]*easyrpc_discovery.RpcLoadBalanceClient)
	pool          *easyrpc_discovery.ConnPool
}

func (s *ConsulManager) InitRegister(
	serviceName string,
	address string,
	port int,
	duration time.Duration) {
	var serviceId = serviceName + ":" + strconv.Itoa(port)
	var client = s.CreateConsulApiClient(s.ConsulAddress)
	var reg = CreateAgentServiceRegistration(TCP, serviceId, serviceName, address, port, fmt.Sprint(duration.Seconds()))
	s.client = client
	s.reg = reg
}

func (s *ConsulManager) DoRegister() {
	s.DoRegisterConsul(s.reg, s.client)
}

func (s *ConsulManager) InitServiceFetcher(
	manager *easyrpc_discovery.RpcServiceManager,
	clearFunc func(m map[string]*easyrpc_discovery.RpcLoadBalanceClient),
	pool *easyrpc_discovery.ConnPool) {
	s.manager = manager
	s.clearFunc = &clearFunc
	s.pool = pool
}

func (s *ConsulManager) DoFetch() {
	var newService map[string]*easyrpc_discovery.AgentService
	newServiceList, e := s.client.Agent().Services()
	if e != nil {
		return
	}
	if newServiceList == nil {
		return
	}
	newService = map[string]*easyrpc_discovery.AgentService{}
	for k, _ := range newServiceList {
		if !strings.Contains(k, "Service") {
			delete(newServiceList, k)
		}
	}
	for k, v := range newServiceList {
		newService[k] = &easyrpc_discovery.AgentService{
			Service: v.Service,
			Port:    v.Port,
			Address: v.Address,
		}
	}
	s.manager.SetNewServiceMap(s.manager, newService, *s.clearFunc, s.pool)
}

func CreateAgentServiceRegistration(consulCheckType ConsulCheckType, id string, serviceName string, address string, port int, time string) *consulapi.AgentServiceRegistration {
	fmt.Println("[ConsulManager]start register consul Rpc Service")
	//????????????????????????
	registration := new(consulapi.AgentServiceRegistration)
	registration.Address = address
	registration.Port = port
	registration.ID = id
	registration.Name = serviceName
	registration.Tags = []string{serviceName}

	//??????check???
	check := new(consulapi.AgentServiceCheck)
	if consulCheckType == TCP {
		check.TCP = registration.Address + ":" + strconv.Itoa(registration.Port)
	} else if consulCheckType == HTTP {
		check.HTTP = registration.Address + ":" + strconv.Itoa(registration.Port)
	}
	//???????????? 5s???
	check.Timeout = time + "s"
	//???????????? 5s???
	check.Interval = time + "s"
	//check?????????30??????????????????
	check.DeregisterCriticalServiceAfter = time + "s"
	//??????check?????????
	registration.Check = check
	return registration
}

func (s *ConsulManager) DoRegisterConsul(registration *consulapi.AgentServiceRegistration, client *consulapi.Client) error {
	err := client.Agent().ServiceRegister(registration)
	if err != nil {
		fmt.Println("[ConsulManager]Register Consul Rpc Service error=", err)
	} else {
		fmt.Println("[ConsulManager]Register Consul Rpc Service success.")
	}
	return err
}

func (s *ConsulManager) CreateConsulApiClient(consulAddress string) *consulapi.Client {
	config := consulapi.DefaultConfig()
	config.Address = consulAddress
	client, err := consulapi.NewClient(config)
	if err != nil {
		fmt.Println("[ConsulManager]new consul client error : ", err)
	}
	return client
}
