package pollqueue

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
)

// PriorityPollQueue type defines the base methods used to implement the `heap` library.  https://pkg.go.dev/container/heap
// Also protected by a mutex
type PriorityPollQueue struct {
	priorityQueue []*PollingPoint
	mu            sync.Mutex
}

func (q *PriorityPollQueue) Len() int {
	return len(q.priorityQueue)
}

func (q *PriorityPollQueue) Less(i, j int) bool {
	qLen := len(q.priorityQueue)
	if i >= qLen || j >= qLen {
		return false
	}
	iPriority := q.priorityQueue[i].PollPriority
	iPriorityNum := 0
	switch iPriority {
	case datatype.PriorityASAP:
		iPriorityNum = 0
	case datatype.PriorityHigh:
		iPriorityNum = 1
	case datatype.PriorityNormal:
		iPriorityNum = 2
	case datatype.PriorityLow:
		iPriorityNum = 3
	}
	jPriority := q.priorityQueue[j].PollPriority
	jPriorityNum := 0
	switch jPriority {
	case datatype.PriorityASAP:
		jPriorityNum = 0
	case datatype.PriorityHigh:
		jPriorityNum = 1
	case datatype.PriorityNormal:
		jPriorityNum = 2
	case datatype.PriorityLow:
		jPriorityNum = 3
	}

	if iPriorityNum < jPriorityNum {
		return true
	}
	if iPriorityNum > jPriorityNum {
		return false
	}

	iTimestamp := q.priorityQueue[i].QueueEntryTime
	jTimestamp := q.priorityQueue[j].QueueEntryTime
	return iTimestamp < jTimestamp
}

func (q *PriorityPollQueue) Swap(i, j int) {
	q.priorityQueue[i], q.priorityQueue[j] = q.priorityQueue[j], q.priorityQueue[i]
}

func (q *PriorityPollQueue) Push(x interface{}) {
	item := x.(*PollingPoint)
	q.priorityQueue = append(q.priorityQueue, item)
}

func (q *PriorityPollQueue) Pop() interface{} {
	old := q.priorityQueue
	n := len(old)
	item := old[n-1]
	q.priorityQueue = old[0 : n-1]
	return item
}

func (q *PriorityPollQueue) getPollingPointIndexByPointUUID(pointUUID string) (*PollingPoint, int) {
	for index, pp := range q.priorityQueue {
		if pp.FFPointUUID == pointUUID {
			return pp, index
		}
	}
	return nil, -1
}

func (q *PriorityPollQueue) GetPollingPointIndexByPointUUID(pointUUID string) (*PollingPoint, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.getPollingPointIndexByPointUUID(pointUUID)
}

func (q *PriorityPollQueue) removePollingPoint(index int) *PollingPoint {
	pp := heap.Remove(q, index).(*PollingPoint)
	pp.resetPollingPointTimers()
	return pp
}

func (q *PriorityPollQueue) RemovePollingPointByPointUUID(pointUUID string) *PollingPoint {
	q.mu.Lock()
	defer q.mu.Unlock()
	pp, index := q.getPollingPointIndexByPointUUID(pointUUID)
	if index >= 0 {
		return q.removePollingPoint(index)
	}
	return pp
}

func (q *PriorityPollQueue) RemovePollingPointByDeviceUUID(deviceUUID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		found := false
		for index, pp := range q.priorityQueue {
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

func (q *PriorityPollQueue) RemovePollingPointByNetworkUUID(networkUUID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		found := false
		for index, pp := range q.priorityQueue {
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

func (q *PriorityPollQueue) AddPollingPoint(pp *PollingPoint) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, index := q.getPollingPointIndexByPointUUID(pp.FFPointUUID)
	if index == -1 {
		heap.Push(q, pp)
		return true
	}
	return false
}

func (q *PriorityPollQueue) UpdatePollingPointByPointUUID(pointUUID string, newPriority datatype.PollPriority) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	pp, index := q.getPollingPointIndexByPointUUID(pointUUID)
	if index >= 0 {
		pp.PollPriority = newPriority
		heap.Fix(q, index)
		return true
	}
	return false
}

func (q *PriorityPollQueue) EmptyQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.Len() > 0 {
		pp := heap.Pop(q).(*PollingPoint)
		pp.resetPollingPointTimers()
	}
}

func (q *PriorityPollQueue) GetNextPollingPoint() (*PollingPoint, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.Len() > 0 {
		pp := heap.Pop(q).(*PollingPoint)
		return pp, nil
	}
	return nil, errors.New("PriorityPollQueue is not enabled")
}
