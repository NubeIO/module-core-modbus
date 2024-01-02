package writemode

import (
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
)

func SetPriorityArrayModeBasedOnWriteMode(pnt *model.Point) bool {
	switch pnt.WriteMode {
	case datatype.ReadOnce, datatype.ReadOnly:
		pnt.PointPriorityArrayMode = datatype.ReadOnlyNoPriorityArrayRequired
		return true
	case datatype.WriteOnce, datatype.WriteOnceReadOnce, datatype.WriteAlways, datatype.WriteOnceThenRead, datatype.WriteAndMaintain:
		pnt.PointPriorityArrayMode = datatype.PriorityArrayToWriteValue
		return true
	}
	return false
}

func IsWriteable(writeMode datatype.WriteMode) bool {
	switch writeMode {
	case datatype.ReadOnce, datatype.ReadOnly:
		return false
	case datatype.WriteOnce, datatype.WriteOnceReadOnce, datatype.WriteAlways, datatype.WriteOnceThenRead, datatype.WriteAndMaintain:
		return true
	default:
		return false
	}
}

func ResetWriteableProperties(point *model.Point) *model.Point {
	point.WriteValueOriginal = nil
	point.WriteValue = nil
	point.WritePriority = nil
	point.CurrentPriority = nil
	point.EnableWriteable = boolean.NewFalse()
	point.WritePollRequired = boolean.NewFalse()
	return point
}
