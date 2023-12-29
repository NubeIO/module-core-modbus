package pkg

import (
	"container/heap"
	"fmt"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/patrickmn/go-cache"
	"time"
)

type Module struct {
	basePath            string
	config              *Config
	dbHelper            nmodule.DBHelper
	enabled             bool
	fault               bool
	grpcMarshaller      nmodule.Marshaller
	moduleName          string
	networks            []*model.Network
	NetworkPollManagers []*pollqueue.NetworkPollManager
	pluginUUID          string
	pollingCancel       func()
	pollingEnabled      bool
	running             bool
	store               *cache.Cache
}

func (m *Module) Init(dbHelper nmodule.DBHelper, moduleName string) error {
	grpcMarshaller := nmodule.GRPCMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.moduleName = moduleName
	m.grpcMarshaller = &grpcMarshaller
	return nil
}

func (m *Module) GetInfo() (*nmodule.Info, error) {
	return &nmodule.Info{
		Name:       m.moduleName,
		Author:     "Nube iO",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}

func NewPollManager(
	conf *pollqueue.Config,
	marshaller nmodule.Marshaller,
	ffNetworkUUID, ffNetworkName, ffPluginUUID, pluginName string,
	maxPollRate float64,
) *pollqueue.NetworkPollManager {
	queue := make([]*pollqueue.PollingPoint, 0)
	pq := &pollqueue.PriorityPollQueue{PriorityQueue: queue}
	heap.Init(pq)

	refQueue := make([]*pollqueue.PollingPoint, 0)
	rq := &pollqueue.PriorityPollQueue{PriorityQueue: refQueue}
	heap.Init(rq)

	outstandingQueue := make([]*pollqueue.PollingPoint, 0)
	opq := &pollqueue.PriorityPollQueue{PriorityQueue: outstandingQueue}
	heap.Init(opq)

	adl := make([]string, 0)
	pqu := &pollqueue.QueueUnloader{
		NextPollPoint:   nil,
		NextUnloadTimer: nil,
		CancelChannel:   nil,
	}
	puwp := make(map[string]bool)
	npq := &pollqueue.NetworkPriorityPollQueue{
		Config:                    conf,
		PriorityQueue:             pq,
		StandbyPollingPoints:      rq,
		OutstandingPollingPoints:  opq,
		PointsUpdatedWhilePolling: puwp,
		QueueUnloader:             pqu,
		FFPluginUUID:              ffPluginUUID,
		FFNetworkUUID:             ffNetworkUUID,
		ActiveDevicesList:         adl,
	}
	pm := new(pollqueue.NetworkPollManager)
	pm.Enable = false
	pm.Config = conf
	pm.PollQueue = npq
	pm.PluginQueueUnloader = nil
	pm.Marshaller = marshaller
	pm.MaxPollRate, _ = time.ParseDuration(fmt.Sprintf("%fs", maxPollRate))
	pm.FFNetworkUUID = ffNetworkUUID
	pm.NetworkName = ffNetworkName
	pm.FFPluginUUID = ffPluginUUID
	pm.PluginName = pluginName
	pm.ASAPPriorityMaxCycleTime, _ = time.ParseDuration("2m")
	pm.HighPriorityMaxCycleTime, _ = time.ParseDuration("5m")
	pm.NormalPriorityMaxCycleTime, _ = time.ParseDuration("15m")
	pm.LowPriorityMaxCycleTime, _ = time.ParseDuration("60m")
	return pm
}
