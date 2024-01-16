package pollqueue

import (
	"container/heap"
	"errors"
	"fmt"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
)

// REFS:
//  - GOLANG HEAP https://pkg.go.dev/container/heap
//  - Worker Queue tutorial: https://www.opsdash.com/blog/job-queues-in-go.html

func (pm *NetworkPollManager) GetNextPollingPoint() *PollingPoint {
	return pm.PollQueue.GetNextPollingPoint()
}

func (pm *NetworkPollManager) RebuildPollingQueue() error {
	pm.pollQueueDebugMsg("RebuildPollingQueue()")
	pm.StopPolling()
	net, err := pm.Marshaller.GetNetwork(pm.FFNetworkUUID, &nmodule.Opts{Args: &nargs.Args{WithDevices: true, WithPoints: true}})
	if err != nil || net.Devices == nil || len(net.Devices) == 0 {
		pm.pollQueueDebugMsg("RebuildPollingQueue() couldn't find any devices for the network %s", pm.FFNetworkUUID)
		return errors.New(fmt.Sprintf("NetworkPollManager.RebuildPollingQueue: couldn't find any devices for the network %s", pm.FFNetworkUUID))
	}
	devs := net.Devices
	for _, dev := range devs {
		if boolean.IsFalse(dev.Enable) {
			continue
		}
		for _, pnt := range dev.Points {
			if boolean.IsFalse(pnt.Enable) {
				continue
			}
			pp := NewPollingPoint(pnt.UUID, pnt.DeviceUUID, dev.NetworkUUID)
			pp.PollPriority = pnt.PollPriority
			pm.pollQueueDebugMsg(fmt.Sprintf("RebuildPollingQueue() pp: %+v", pp))
			if PollOnStartCheck(pnt) {
				pm.AddToPriorityQueue(pp)
			} else {
				pm.PollingPointCompleteNotification(pp, pnt, true, true, 0, true, false, NORMAL_RETRY, true)
			}
		}
	}
	heap.Init(pm.PollQueue.PriorityQueue)
	return nil
}

func (pm *NetworkPollManager) PollingPointCompleteNotification(pp *PollingPoint, point *model.Point, writeSuccess, readSuccess bool, pollTimeSecs float64, pointUpdate, resetToConfiguredPriority bool, retryType PollRetryType, pollingWasNotRequired bool) {
	pm.pollQueuePollingMsg(fmt.Sprintf("POLLING COMPLETE: Point UUID: %s, writeSuccess: %t, readSuccess: %t, pointUpdate: %t, pollingWasNotRequired: %t, retryType: %s, pollTime: %f", pp.FFPointUUID, writeSuccess, readSuccess, pointUpdate, pollingWasNotRequired, retryType, pollTimeSecs))

	if !pointUpdate {
		// This will update the relevant PollManager statistics
		pm.PollCompleteStatsUpdate(pp, pollTimeSecs)
	}

	// Reset poll priority to set value (in cases where pp has been escalated to ASAP).
	if resetToConfiguredPriority {
		pp.PollPriority = point.PollPriority
	}

	pp.resetPollingPointTimers()

	// point was deleted while it was out for polling
	if pm.PollQueue.QueueUnloader.RemoveCurrent && pm.PollQueue.QueueUnloader.CurrentPollPoint.FFPointUUID == pp.FFPointUUID {
		pm.PollQueue.QueueUnloader.RemoveCurrent = false
		return
	}

	// instantly re-add if it was updated while polling
	val, ok := pm.PollQueue.PointsUpdatedWhilePolling[point.UUID]
	if ok {
		delete(pm.PollQueue.PointsUpdatedWhilePolling, point.UUID)
		if val == true { // point needs an ASAP write
			pp.PollPriority = datatype.PriorityASAP
			pm.AddToPriorityQueue(pp)
			return
		}
	}

	// used to avoid a db write if neither of these are changed
	origReadPollReq := *point.ReadPollRequired
	origWritePollReq := *point.WritePollRequired

	addSuccess := true

	switch point.WriteMode {
	case datatype.ReadOnce: // If read_successful then don't re-add.
		point.WritePollRequired = boolean.NewFalse()
		if retryType == NEVER_RETRY || ((readSuccess || pollingWasNotRequired) && retryType == NORMAL_RETRY) {
			point.ReadPollRequired = boolean.NewFalse()
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		} else if (boolean.IsTrue(point.ReadPollRequired) && !readSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			point.ReadPollRequired = boolean.NewTrue()
			pm.AddToPriorityQueue(pp)
		} else if retryType == DELAYED_RETRY {
			point.ReadPollRequired = boolean.NewTrue()
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
		}

	case datatype.ReadOnly: // Re-add with ReadPollRequired true, WritePollRequired false.
		point.WritePollRequired = boolean.NewFalse()
		point.ReadPollRequired = boolean.NewTrue()
		if ((readSuccess || pollingWasNotRequired) && retryType == NORMAL_RETRY) || retryType == DELAYED_RETRY {
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
		} else if (!readSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			pm.AddToPriorityQueue(pp)
		} else if retryType == NEVER_RETRY {
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		}

	case datatype.WriteOnce: // If write_successful then don't re-add.
		point.ReadPollRequired = boolean.NewFalse()
		if ((writeSuccess || pollingWasNotRequired) && retryType == NORMAL_RETRY) || retryType == NEVER_RETRY {
			point.WritePollRequired = boolean.NewFalse()
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		} else if (boolean.IsTrue(point.WritePollRequired) && !writeSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			point.WritePollRequired = boolean.NewTrue() // TODO: this might cause these points to write more than once.
			pm.AddToPriorityQueue(pp)
		} else if retryType == DELAYED_RETRY {
			point.WritePollRequired = boolean.NewTrue()
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
		}

	case datatype.WriteOnceReadOnce: // If write_successful and read_success then don't re-add.
		if (boolean.IsTrue(point.WritePollRequired) && writeSuccess && retryType == NORMAL_RETRY) || retryType == NEVER_RETRY {
			point.WritePollRequired = boolean.NewFalse()
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		} else if pointUpdate || (boolean.IsTrue(point.WritePollRequired) && !writeSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			point.WritePollRequired = boolean.NewTrue()
			if pointUpdate {
				point.ReadPollRequired = boolean.NewTrue()
			}
			pm.AddToPriorityQueue(pp)
			break
		} else if retryType == DELAYED_RETRY {
			point.WritePollRequired = boolean.NewTrue()
			point.ReadPollRequired = boolean.NewTrue()
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
			break
		}
		if readSuccess && retryType == NORMAL_RETRY || retryType == NEVER_RETRY {
			point.ReadPollRequired = boolean.NewFalse()
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		} else if boolean.IsTrue(point.ReadPollRequired) && !readSuccess && retryType == NORMAL_RETRY || retryType == IMMEDIATE_RETRY {
			point.ReadPollRequired = boolean.NewTrue()
			pm.AddToPriorityQueue(pp)
		}

	case datatype.WriteAlways: // Re-add with ReadPollRequired false, WritePollRequired true. confirm that a successful write ensures the value is set to the write value.
		point.ReadPollRequired = boolean.NewFalse()
		point.WritePollRequired = boolean.NewTrue()
		if ((writeSuccess || pollingWasNotRequired) && retryType == NORMAL_RETRY) || retryType == DELAYED_RETRY {
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
		} else if (!writeSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			pm.AddToPriorityQueue(pp)
		} else if retryType == NEVER_RETRY {
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		}

	case datatype.WriteOnceThenRead: // If write_successful: Re-add with ReadPollRequired true, WritePollRequired false.
		point.ReadPollRequired = boolean.NewTrue()
		if retryType == NEVER_RETRY {
			if writeSuccess {
				point.WritePollRequired = boolean.NewFalse()
			}
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
		} else if pointUpdate || (boolean.IsTrue(point.WritePollRequired) && !writeSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			if writeSuccess {
				point.WritePollRequired = boolean.NewFalse()
			}
			pm.AddToPriorityQueue(pp)
			break
		} else if (boolean.IsTrue(point.WritePollRequired) && writeSuccess && retryType == NORMAL_RETRY) || retryType == DELAYED_RETRY {
			if writeSuccess {
				point.WritePollRequired = boolean.NewFalse()
			}
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
			break
		}
		if readSuccess && retryType == NORMAL_RETRY || retryType == DELAYED_RETRY {
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
			break
		} else if !readSuccess && retryType == NORMAL_RETRY || retryType == IMMEDIATE_RETRY {
			pm.AddToPriorityQueue(pp)
		}

	case datatype.WriteAndMaintain: // If write_successful: Re-add with ReadPollRequired true, WritePollRequired false.  Need to check that write value matches present value after each read poll.
		point.ReadPollRequired = boolean.NewTrue()
		if (boolean.IsTrue(point.WritePollRequired) && !writeSuccess && retryType == NORMAL_RETRY) || retryType == IMMEDIATE_RETRY {
			pm.AddToPriorityQueue(pp)
			break
		} else if retryType == DELAYED_RETRY {
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
			break
		} else if retryType == NEVER_RETRY {
			addSuccess = pm.PollQueue.AddToStandbyQueue(pp)
			break
		}

		if point.WriteValue != nil {
			noPV := true
			var readValue float64
			if point.PresentValue != nil {
				noPV = false
				readValue = *point.PresentValue
			}
			if noPV || readValue != *point.WriteValue {
				point.WritePollRequired = boolean.NewTrue()
				pm.AddToPriorityQueue(pp)
			} else {
				point.WritePollRequired = boolean.NewFalse()
				addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
			}
		} else {
			// If WriteValue is nil we still need to re-add the point to perform a read
			point.WritePollRequired = boolean.NewFalse()
			addSuccess = pm.AddToStandbyQueueWithRePoll(pp, point)
		}
	}

	if !addSuccess {
		pm.pollQueueErrorMsg(fmt.Sprintf("Modbus PollingPointCompleteNotification(): polling point could not be added to StandbyPollingPoints slice.  (%s)", pp.FFPointUUID))
	}

	pm.PollQueue.QueueUnloader.CurrentPollPoint = nil

	if *point.ReadPollRequired != origReadPollReq || *point.WritePollRequired != origWritePollReq {
		point, _ = pm.Marshaller.UpdatePoint(point.UUID, point)
	}
	pm.pollQueuePollingMsg("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
}

func (pm *NetworkPollManager) AddToPriorityQueue(pp *PollingPoint) {
	pp.LockupAlertTimer = pm.MakeLockupTimerFunc(pp.PollPriority)
	pm.PollQueue.AddToPriorityQueue(pp)
}

func (pm *NetworkPollManager) AddToStandbyQueueWithRePoll(pp *PollingPoint, point *model.Point) bool {
	duration := pm.GetPollRateDuration(point.PollRate, pp.FFDeviceUUID)
	pp.RepollTimer = time.AfterFunc(duration, pm.MakePollingPointRepollCallback(pp))
	return pm.PollQueue.AddToStandbyQueue(pp)
}

func (pm *NetworkPollManager) MakePollingPointRepollCallback(pp *PollingPoint) func() {
	f := func() {
		pp.RepollTimer = nil
		ppOld := pm.PollQueue.RemoveFromStandbyQueue(pp)
		if ppOld == nil {
			pm.pollQueueErrorMsg(fmt.Sprintf("Modbus MakePollingPointRepollCallback(): polling point could not be found in StandbyPollingPoints.  (%s)", pp.FFPointUUID))
		}
		pm.AddToPriorityQueue(pp)
	}
	return f
}

func (pm *NetworkPollManager) MakeLockupTimerFunc(priority datatype.PollPriority) *time.Timer {
	timeoutDuration := 5 * time.Minute

	switch priority {
	case datatype.PriorityASAP:
		timeoutDuration = pm.ASAPPriorityMaxCycleTime

	case datatype.PriorityHigh:
		timeoutDuration = pm.HighPriorityMaxCycleTime

	case datatype.PriorityNormal:
		timeoutDuration = pm.NormalPriorityMaxCycleTime

	case datatype.PriorityLow:
		timeoutDuration = pm.LowPriorityMaxCycleTime
	}

	f := func() {
		pm.pollQueueDebugMsg("Polling Lockout Timer Expired! Polling Priority: %d,  Polling Network: %s", priority, pm.FFNetworkUUID)
		name := "unknown"
		plugin, err := pm.Marshaller.GetPlugin(pm.FFPluginUUID)
		if plugin != nil && err == nil {
			name = plugin.Name
		}
		switch priority {
		case datatype.PriorityASAP:
			pm.Statistics.ASAPPriorityLockupAlert = true
		case datatype.PriorityHigh:
			pm.Statistics.HighPriorityLockupAlert = true
		case datatype.PriorityNormal:
			pm.Statistics.NormalPriorityLockupAlert = true
		case datatype.PriorityLow:
			pm.Statistics.LowPriorityLockupAlert = true
		}
		pm.pollQueueErrorMsg(fmt.Sprintf("%s Plugin: %s Priority Poll Queue LOCKUP", name, priority))
	}
	return time.AfterFunc(timeoutDuration, f)
}
