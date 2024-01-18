package pkg

import (
	"context"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-modbus/pollqueue"
	"github.com/NubeIO/module-core-modbus/smod"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/patrickmn/go-cache"
)

type Module struct {
	basePath            string
	config              *Config
	dbHelper            nmodule.DBHelper
	grpcMarshaller      nmodule.Marshaller
	moduleName          string
	networks            []*model.Network
	NetworkPollManagers []*pollqueue.NetworkPollManager
	pluginUUID          string
	pollingContext      context.Context
	pollingCancel       func()
	pollingEnabled      bool
	running             bool
	store               *cache.Cache
	mbClients           map[string]*smod.ModbusClient
}

func (m *Module) Init(dbHelper nmodule.DBHelper, moduleName string) error {
	InitRouter()
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

func (m *Module) setUUID() {
	q, err := m.grpcMarshaller.GetPluginByName(m.moduleName)
	if err != nil {
		return
	}
	m.pluginUUID = q.UUID
}
