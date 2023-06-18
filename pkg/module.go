package pkg

import (
	"github.com/NubeIO/rubix-os/module/shared"
	"github.com/patrickmn/go-cache"
)

type Module struct {
	basePath       string
	config         *Config
	dbHelper       shared.DBHelper
	enabled        bool
	fault          bool
	grpcMarshaller shared.Marshaller
	moduleName     string
	pluginUUID     string
	running        bool
	store          *cache.Cache
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
