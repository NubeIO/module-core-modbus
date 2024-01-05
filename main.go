package main

import (
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-modbus/pkg"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: nmodule.HandshakeConfig,
		Plugins:         plugin.PluginSet{"module-core-modbus": &nmodule.NubeModule{Impl: &pkg.Module{}}},
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func main() {
	ServePlugin()
}
