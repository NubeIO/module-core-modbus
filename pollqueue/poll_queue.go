package pollqueue

import (
	"container/heap"
	"fmt"
	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	log "github.com/sirupsen/logrus"
	"time"
)

type NetworkPriorityPollQueue struct {
	Config                    *Config
	PriorityQueue             *PriorityPollQueue // what all polling points are drawn from
	StandbyPollingPoints      *StandbyPollQueue  // contains polling points that are NOT in the active polling queue, it is mostly a reference so that we can periodically find out if any points have been dropped from polling.
	PointsUpdatedWhilePolling map[string]bool    // UUIDs of points that have been updated while they were out for polling.  bool is true if the point needs to be written ASAP
	QueueUnloader             QueueUnloader
}

type QueueUnloader struct {
	NextPollPoint    *PollingPoint
	CurrentPollPoint *PollingPoint // TODO: turn this back into a list later
	RemoveCurrent    bool
}

func NewNetworkPriorityPollQueue(config *Config) *NetworkPriorityPollQueue {
	queue := &PriorityPollQueue{priorityQueue: make([]*PollingPoint, 0)}
	heap.Init(queue)
	return &NetworkPriorityPollQueue{
		Config:                    config,
		PriorityQueue:             queue,
		StandbyPollingPoints:      &StandbyPollQueue{},
		PointsUpdatedWhilePolling: make(map[string]bool),
	}
}

func (nq *NetworkPriorityPollQueue) AddToPriorityQueue(pp *PollingPoint) bool {
	nq.pollQueueDebugMsg("NetworkPriorityPollQueue AddToPriorityQueue(): ", pp.FFPointUUID)
	// TODO: this should be removed once no errors detected
	if _, index := nq.PriorityQueue.GetPollingPointIndexByPointUUID(pp.FFPointUUID); index != -1 {
		log.Errorf("NetworkPriorityPollQueue.AddToPriorityQueue: PollingPoint %s already exists in polling queue.", pp.FFPointUUID)
		return false
	}

	pp.QueueEntryTime = time.Now().Unix()
	return nq.PriorityQueue.AddPollingPoint(pp)
}

func (nq *NetworkPriorityPollQueue) AddToStandbyQueue(pp *PollingPoint) bool {
	return nq.StandbyPollingPoints.AddPollingPoint(pp)
}

func (nq *NetworkPriorityPollQueue) RemovePollingPointByPointUUID(pointUUID string) *PollingPoint {
	nq.pollQueueDebugMsg("RemovePollingPointByPointUUID(): ", pointUUID)
	var pp *PollingPoint = nil
	if nq.QueueUnloader.CurrentPollPoint != nil && nq.QueueUnloader.CurrentPollPoint.FFPointUUID == pointUUID {
		pp = nq.QueueUnloader.CurrentPollPoint // don't set to nil as the complete notification func needs to check
		nq.QueueUnloader.RemoveCurrent = true
	} else if nq.QueueUnloader.NextPollPoint != nil && nq.QueueUnloader.NextPollPoint.FFPointUUID == pointUUID {
		pp = nq.QueueUnloader.NextPollPoint
		nq.QueueUnloader.NextPollPoint = nil
		nq.setNextPollPoint()
	} else {
		pp = nq.PriorityQueue.RemovePollingPointByPointUUID(pointUUID)
		if pp == nil {
			pp = nq.StandbyPollingPoints.RemovePollingPointByPointUUID(pointUUID)
		}
	}
	return pp
}

func (nq *NetworkPriorityPollQueue) RemoveFromStandbyQueue(pp *PollingPoint) *PollingPoint {
	return nq.StandbyPollingPoints.RemovePollingPointByPointUUID(pp.FFPointUUID)
}

func (nq *NetworkPriorityPollQueue) RemovePollingPointByDeviceUUID(deviceUUID string) bool {
	nq.pollQueueDebugMsg("RemovePollingPointByDeviceUUID(): ", deviceUUID)
	nq.PriorityQueue.RemovePollingPointByDeviceUUID(deviceUUID)
	nq.StandbyPollingPoints.RemovePollingPointByDeviceUUID(deviceUUID)
	return true
}

func (nq *NetworkPriorityPollQueue) UpdatePollingPointByPointUUID(pointUUID string, newPriority datatype.PollPriority) bool {
	found := nq.PriorityQueue.UpdatePollingPointByPointUUID(pointUUID, newPriority)
	if !found {
		found = nq.StandbyPollingPoints.UpdatePollingPointByPointUUID(pointUUID, newPriority)
	}
	return found
}

func (nq *NetworkPriorityPollQueue) GetPollingPointByPointUUID(pointUUID string) *PollingPoint {
	nq.pollQueueDebugMsg("NetworkPriorityPollQueue GetPollingPointByPointUUID(): ", pointUUID)
	if nq.QueueUnloader.CurrentPollPoint != nil && nq.QueueUnloader.CurrentPollPoint.FFPointUUID == pointUUID {
		return nq.QueueUnloader.CurrentPollPoint
	}
	if nq.QueueUnloader.NextPollPoint != nil && nq.QueueUnloader.NextPollPoint.FFPointUUID == pointUUID {
		return nq.QueueUnloader.NextPollPoint
	}
	pp, index := nq.PriorityQueue.GetPollingPointIndexByPointUUID(pointUUID)
	if index != -1 {
		return pp
	}
	pp, index = nq.StandbyPollingPoints.GetPollingPointIndexByPointUUID(pointUUID)
	if index != -1 {
		return pp
	}

	return nil
}

func (nq *NetworkPriorityPollQueue) GetNextPollingPoint() *PollingPoint {
	pp := nq.QueueUnloader.NextPollPoint
	nq.QueueUnloader.CurrentPollPoint = pp
	nq.QueueUnloader.NextPollPoint = nil
	nq.setNextPollPoint()
	return pp
}

func (nq *NetworkPriorityPollQueue) Start() {
	nq.QueueUnloader = QueueUnloader{nil, nil, false}
	nq.setNextPollPoint()
}

func (nq *NetworkPriorityPollQueue) Stop() {
	nq.EmptyQueue()
}

func (nq *NetworkPriorityPollQueue) EmptyQueue() {
	nq.PriorityQueue.EmptyQueue()
	nq.StandbyPollingPoints.EmptyQueue()
	if nq.QueueUnloader.NextPollPoint != nil {
		nq.QueueUnloader.NextPollPoint.resetPollingPointTimers()
		nq.QueueUnloader.NextPollPoint = nil
	}
	if nq.QueueUnloader.CurrentPollPoint != nil {
		nq.QueueUnloader.CurrentPollPoint.resetPollingPointTimers()
		nq.QueueUnloader.CurrentPollPoint = nil
	}
}

func (nq *NetworkPriorityPollQueue) setNextPollPoint() {
	if nq.QueueUnloader.NextPollPoint != nil {
		return
	}
	pp, err := nq.PriorityQueue.GetNextPollingPoint()
	if pp != nil && err == nil {
		nq.QueueUnloader.NextPollPoint = pp
		pp.resetPollingPointTimers()
	}
}

func (nq *NetworkPriorityPollQueue) pollQueueDebugMsg(args ...interface{}) {
	if nstring.IsEqualIgnoreCase(nq.Config.LogLevel, "DEBUG") {
		prefix := "Poll Queue: "
		log.Info(prefix, fmt.Sprint(args...))
	}
}
func (nq *NetworkPriorityPollQueue) pollQueueErrorMsg(args ...interface{}) {
	prefix := "Poll Queue: "
	log.Error(prefix, fmt.Sprint(args...))
}
