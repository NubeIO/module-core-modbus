package pollqueue

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"time"
)

type PollingPoint struct {
	PollPriority     datatype.PollPriority
	FFPointUUID      string
	FFDeviceUUID     string
	FFNetworkUUID    string
	RepollTimer      *time.Timer
	QueueEntryTime   int64
	LockupAlertTimer *time.Timer
}

func (pp *PollingPoint) resetPollingPointTimers() {
	if pp.RepollTimer != nil {
		pp.RepollTimer.Stop()
		pp.RepollTimer = nil
	}
	if pp.LockupAlertTimer != nil {
		pp.LockupAlertTimer.Stop()
		pp.LockupAlertTimer = nil
	}
}

func NewPollingPoint(ffPointUUID, ffDeviceUUID, ffNetworkUUID string) *PollingPoint {
	pp := &PollingPoint{datatype.PriorityNormal, ffPointUUID, ffDeviceUUID, ffNetworkUUID, nil, 0, nil}
	return pp
}

func NewPollingPointWithPriority(ffPointUUID, ffDeviceUUID, ffNetworkUUID string, priority datatype.PollPriority) *PollingPoint {
	pp := &PollingPoint{priority, ffPointUUID, ffDeviceUUID, ffNetworkUUID, nil, 0, nil}
	return pp
}
