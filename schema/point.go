package schema

import (
	"github.com/NubeIO/lib-schema-go/schema"
	"github.com/NubeIO/lib-units/units"
)

type PointSchema struct {
	UUID        schema.UUID        `json:"uuid"`
	Name        schema.Name        `json:"name"`
	Description schema.Description `json:"description"`
	Enable      schema.Enable      `json:"enable"`

	ObjectTypeModbus ObjectTypeModbus `json:"object_type"`
	AddressId        struct {
		Type        string `json:"type" default:"number"`
		Title       string `json:"title" default:"Register"`
		Default     int    `json:"default" default:"1"`
		Minimum     int    `json:"minimum" default:"1"`
		Maximim     int    `json:"maximum" default:"9999"`
		ReadOnly    bool   `json:"readOnly" default:"false"`
		Description string `json:"description" default:"Decimal format: 1-9999"`
	} `json:"address_id"`
	DataType       DataType         `json:"data_type"`
	WriteMode      schema.WriteMode `json:"write_mode"`
	ObjectEncoding ObjectEncoding   `json:"object_encoding"`

	PollPriority schema.PollPriority `json:"poll_priority"`
	PollRate     schema.PollRate     `json:"poll_rate"`

	MultiplicationFactor schema.MultiplicationFactor `json:"multiplication_factor"`
	ScaleEnable          schema.ScaleEnable          `json:"scale_enable"`
	ScaleInMin           schema.ScaleInMin           `json:"scale_in_min"`
	ScaleInMax           schema.ScaleInMax           `json:"scale_in_max"`
	ScaleOutMin          schema.ScaleOutMin          `json:"scale_out_min"`
	ScaleOutMax          schema.ScaleOutMax          `json:"scale_out_max"`
	Offset               schema.Offset               `json:"offset"`
	Decimal              schema.Decimal              `json:"decimal"`
	Fallback             schema.Fallback             `json:"fallback"`

	Unit schema.MeasurementUnit `json:"unit"`

	HistoryEnable       schema.HistoryEnableDefaultTrue `json:"history_enable"`
	HistoryType         schema.HistoryType              `json:"history_type"`
	HistoryInterval     schema.HistoryInterval          `json:"history_interval"`
	HistoryCOVThreshold schema.HistoryCOVThreshold      `json:"history_cov_threshold"`
}

func GetPointSchema() *PointSchema {
	m := &PointSchema{}
	m.Unit.Options, m.Unit.EnumName = units.SupportedUnitsNames()
	schema.Set(m)
	return m
}
