package pollqueue

import (
	"sync"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
)

// StandbyPollQueue type is a standard slice protected by a mutex
type StandbyPollQueue struct {
	queue []*PollingPoint
	mu    sync.Mutex
}

func (q *StandbyPollQueue) Len() int {
	return len(q.queue)
}

func (q *StandbyPollQueue) getPollingPointIndexByPointUUID(pointUUID string) (*PollingPoint, int) {
	for index, pp := range q.queue {
		if pp.FFPointUUID == pointUUID {
			return pp, index
		}
	}
	return nil, -1
}

func (q *StandbyPollQueue) GetPollingPointIndexByPointUUID(pointUUID string) (*PollingPoint, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.getPollingPointIndexByPointUUID(pointUUID)
}

func (q *StandbyPollQueue) removePollingPoint(index int) *PollingPoint {
	l := len(q.queue) - 1
	pp := q.queue[index]
	q.queue[index] = q.queue[l]
	q.queue = q.queue[:l]
	pp.resetPollingPointTimers()
	return pp
}

func (q *StandbyPollQueue) RemovePollingPointByPointUUID(pointUUID string) *PollingPoint {
	q.mu.Lock()
	defer q.mu.Unlock()
	pp, index := q.getPollingPointIndexByPointUUID(pointUUID)
	if index >= 0 {
		return q.removePollingPoint(index)
	}
	return pp
}

func (q *StandbyPollQueue) RemovePollingPointByDeviceUUID(deviceUUID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		found := false
		for index, pp := range q.queue {
			if pp.FFDeviceUUID == deviceUUID {
				q.removePollingPoint(index)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return true
}

func (q *StandbyPollQueue) RemovePollingPointByNetworkUUID(networkUUID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		found := false
		for index, pp := range q.queue {
			if pp.FFNetworkUUID == networkUUID {
				q.removePollingPoint(index)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return true
}

func (q *StandbyPollQueue) AddPollingPoint(pp *PollingPoint) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = append(q.queue, pp)
	return true
}

func (q *StandbyPollQueue) UpdatePollingPointByPointUUID(pointUUID string, newPriority datatype.PollPriority) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	pp, index := q.getPollingPointIndexByPointUUID(pointUUID)
	if index >= 0 {
		pp.PollPriority = newPriority
		return true
	}
	return false
}

func (q *StandbyPollQueue) EmptyQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, pp := range q.queue {
		pp.resetPollingPointTimers()
	}
	q.queue = nil
}
