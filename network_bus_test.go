package evbus

import (
	"fmt"
	"testing"
)

func TestNewServer(t *testing.T) {
	serverBus := NewServer(":2010", "/_server_bus_", New())
	serverBus.Start()
	if serverBus == nil || !serverBus.service.started {
		t.Log("New server EventBus not created!")
		t.Fail()
	}
	serverBus.Stop()
}

func TestNewClient(t *testing.T) {
	clientBus := NewClient(":2015", "/_client_bus_", New())
	clientBus.Start()
	if clientBus == nil || !clientBus.service.started {
		t.Log("New client EventBus not created!")
		t.Fail()
	}
	clientBus.Stop()
}

func TestRegister(t *testing.T) {
	serverPath := "/_server_bus_"
	serverBus := NewServer(":2010", serverPath, New())

	args := &SubscribeArg{serverBus.address, serverPath, PublishService, Subscribe, "topic"}
	reply := new(bool)

	serverBus.service.Register(args, reply)

	if serverBus.eventBus.HasCallback("topic_topic") {
		t.Fail()
	}
	if !serverBus.eventBus.HasCallback("topic") {
		t.Fail()
	}
}

func TestPushEvent(t *testing.T) {
	clientBus := NewClient("localhost:2015", "/_client_bus_", New())

	eventArgs := make([]interface{}, 1)
	eventArgs[0] = 10

	clientArg := &ClientArg{eventArgs, "topic"}
	reply := new(bool)

	fn := func(a int) {
		if a != 10 {
			t.Fail()
		}
	}

	clientBus.eventBus.Subscribe("topic", fn)
	clientBus.service.PushEvent(clientArg, reply)
	if !(*reply) {
		t.Fail()
	}
}

func TestServerPublish(t *testing.T) {
	serverBus := NewServer(":2020", "/_server_bus_b", New())
	serverBus.Start()

	fn := func(a int) {
		if a != 10 {
			t.Fail()
		}
	}

	clientBus := NewClient(":2025", "/_client_bus_b", New())
	clientBus.Start()

	clientBus.Subscribe("topic", fn, ":2010", "/_server_bus_b")

	serverBus.EventBus().Publish("topic", 10)

	clientBus.Stop()
	serverBus.Stop()
}

func TestNetworkBus(t *testing.T) {
	networkBusA := NewNetworkBus(":2035", "/_net_bus_A")
	networkBusA.Start()

	networkBusB := NewNetworkBus(":2030", "/_net_bus_B")
	networkBusB.Start()

	fnA := func(a int) {
		if a != 10 {
			t.Fail()
		}
	}
	networkBusA.Subscribe("topic-A", fnA, ":2030", "/_net_bus_B")
	networkBusB.EventBus().Publish("topic-A", 10)

	fnB := func(a int) {
		if a != 20 {
			t.Fail()
		}
	}
	networkBusB.Subscribe("topic-B", fnB, ":2035", "/_net_bus_A")
	networkBusA.EventBus().Publish("topic-B", 20)

	networkBusA.Stop()
	networkBusB.Stop()
}

func Test_Local(t *testing.T) {
	bus := New()

	// 事件
	bus.Subscribe("event", func(a int) {
		fmt.Printf("[sub] %d \n", a)
	})
	bus.Publish("event", 10)

	// 查询
	bus.Subscribe("query", func(result *int) {
		*result = 10
		return
	})
	var result int
	bus.Publish("query", &result)
	fmt.Printf("[query] %d \n", result)
}

func Test_Network(t *testing.T) {
	// 服务端
	serverBus := NewServer(":2040", "/_server_bus_", New())
	serverBus.Start()

	// 客户端
	clientBus := NewClient(":2045", "/_client_bus_", New())
	clientBus.Start()

	// 事件: 服务器推送客户端消费
	clientBus.Subscribe("event", func(a int) {
		fmt.Printf("[sub] %d \n", a)
	}, ":2040", "/_server_bus_")
	serverBus.EventBus().Publish("event", 10)

	// 事件: 客户端推送服务器消费 (编译不通过)
	// serverBus.Subscribe("event", func(a int) {
	// 	fmt.Printf("[sub] %d \n", a)
	// }, ":2045", "/_client_bus_")
	// clientBus.EventBus().Publish("event", 10)

	// 查询 panic 指针类型
	clientBus.Subscribe("query", func(result *int) {
		*result = 10
		return
	}, ":2040", "/_server_bus_")
	var result int
	serverBus.EventBus().Publish("query", &result)
	fmt.Printf("[query] %d \n", result)

	clientBus.Stop()
	serverBus.Stop()
}

func Test_Network2(t *testing.T) {
	var (
		netAAddr = ":2050"
		netAPath = "/_net_bus_A"
		netBAddr = ":2055"
		netBPath = "/_net_bus_B"
	)

	fmt.Printf("Start\n")

	netA := NewNetworkBus(netAAddr, netAPath)
	netA.Start()
	netB := NewNetworkBus(netBAddr, netBPath)
	netB.Start()

	// netA 推送 netB 消费
	netB.Subscribe("event1", func(a int) {
		fmt.Printf("[netA 推送 netB 消费] %d \n", a)
	}, netAAddr, netAPath)
	netA.EventBus().Publish("event1", 10)

	// netB 推送 netA 消费
	netA.Subscribe("event2", func(a int) {
		fmt.Printf("[netB 推送 netA 消费] %d \n", a)
	}, netBAddr, netBPath)
	netB.EventBus().Publish("event2", 10)

	// netA 推送 netA 本地消费
	netA.EventBus().Subscribe("event3", func(a int) {
		fmt.Printf("[netA 推送 netA 本地消费] %d \n", a)
	})
	netA.EventBus().Publish("event3", 10)

	// netA 推送 netA 本地查询
	netA.EventBus().Subscribe("query1", func(result *int) {
		*result = 10
		return
	})
	var result int
	netA.EventBus().Publish("query1", &result)
	fmt.Printf("[netA 推送 netA 本地查询] %d \n", result)

	// netA 推送 netB 查询 panic
	// netB.Subscribe("query2", func(result *int) {
	// 	*result = 10
	// 	return
	// }, netAAddr, netAPath)
	// var result2 int
	// netA.EventBus().Publish("query2", &result2)
	// fmt.Printf("[netA 推送 netB 查询] %d \n", result2)

	netA.Stop()
	netB.Stop()
}
