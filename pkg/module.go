package pkg

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
	"github.com/NubeIO/rubix-os/module/shared"
	"github.com/NubeIO/rubix-os/services/pollqueue"
	"github.com/patrickmn/go-cache"
)

type Module struct {
	// bus                 eventbus.BusService
	// pollingCancel       func()
	basePath            string
	config              *Config
	dbHelper            shared.DBHelper
	enabled             bool
	fault               bool
	grpcMarshaller      shared.Marshaller
	moduleName          string
	networks            []*model.Network
	NetworkPollManagers []*pollqueue.NetworkPollManager
	pluginUUID          string
	pollingEnabled      bool
	running             bool
	store               *cache.Cache
}

func (m *Module) Init(dbHelper shared.DBHelper, moduleName string) error {
	grpcMarshaller := shared.GRPCMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.moduleName = moduleName
	m.grpcMarshaller = &grpcMarshaller
	return nil
}

func (m *Module) GetInfo() (*shared.Info, error) {
	return &shared.Info{
		Name:       name,
		Author:     "Nube iO",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}
