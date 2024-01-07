package pollqueue

import (
	"fmt"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"math"
	"time"
)

type PollStatistics struct {
	MaxPollExecuteTimeSecs        float64 // time in seconds for polling to complete (poll response time, doesn't include the time in queue).
	AveragePollExecuteTimeSecs    float64 // time in seconds for polling to complete (poll response time, doesn't include the time in queue).
	MinPollExecuteTimeSecs        float64 // time in seconds for polling to complete (poll response time, doesn't include the time in queue).
	TotalPollQueueLength          int64   // number of polling points in the current queue.
	TotalStandbyPointsLength      int64   // number of polling points in the standby list.
	TotalPointsOutForPolling      int64   // number of points currently out for polling (currently being handled by the protocol plugin).
	ASAPPriorityPollQueueLength   int64   // number of ASAP priority polling points in the current queue.
	HighPriorityPollQueueLength   int64   // number of High priority polling points in the current queue.
	NormalPriorityPollQueueLength int64   // number of Normal priority polling points in the current queue.
	LowPriorityPollQueueLength    int64   // number of Low priority polling points in the current queue.
	ASAPPriorityAveragePollTime   float64 // average time in seconds between ASAP priority polling point added to current queue, and polling complete.
	HighPriorityAveragePollTime   float64 // average time in seconds between High priority polling point added to current queue, and polling complete.
	NormalPriorityAveragePollTime float64 // average time in seconds between Normal priority polling point added to current queue, and polling complete.
	LowPriorityAveragePollTime    float64 // average time in seconds between Low priority polling point added to current queue, and polling complete.
	TotalPollCount                int64   // total number of polls completed.
	ASAPPriorityPollCount         int64   // total number of ASAP priority polls completed.
	HighPriorityPollCount         int64   // total number of High priority polls completed.
	NormalPriorityPollCount       int64   // total number of Normal priority polls completed.
	LowPriorityPollCount          int64   // total number of Low priority polls completed.
	ASAPPriorityPollCountForAvg   int64   // number of poll times included in avg polling time for ASAP priority (some are excluded because they have been modified while in the queue).
	HighPriorityPollCountForAvg   int64   // number of poll times included in avg polling time for High priority (some are excluded because they have been modified while in the queue).
	NormalPriorityPollCountForAvg int64   // number of poll times included in avg polling time for Normal priority (some are excluded because they have been modified while in the queue).
	LowPriorityPollCountForAvg    int64   // number of poll times included in avg polling time for Low priority (some are excluded because they have been modified while in the queue).
	ASAPPriorityLockupAlert       bool    // alert if poll time has exceeded the ASAPPriorityMaxCycleTime
	HighPriorityLockupAlert       bool    // alert if poll time has exceeded the HighPriorityMaxCycleTime
	NormalPriorityLockupAlert     bool    // alert if poll time has exceeded the NormalPriorityMaxCycleTime
	LowPriorityLockupAlert        bool    // alert if poll time has exceeded the LowPriorityMaxCycleTime
	PollingStartTimeUnix          int64   // unix time (seconds) at polling start time.  Used for calculating Busy Time.
	BusyTime                      float64 // percent of the time that the plugin is actively polling.
	EnabledTime                   float64 // time in seconds that the statistics have been running for.
	PortUnavailableTime           float64 // time in seconds that the serial port has been unavailable.
	PortUnavailableStartTime      int64   // unix time (seconds) when port became unavailable.  Used for calculating downtime.
}

func (pm *NetworkPollManager) GetPollingQueueStatistics() *dto.PollQueueStatistics {
	pm.pollQueueDebugMsg("GetPollingQueueStatistics()")
	stats := dto.PollQueueStatistics{}
	stats.Enable = pm.Enable

	stats.FFNetworkUUID = pm.FFNetworkUUID
	stats.NetworkName = pm.NetworkName
	stats.PluginName = pm.PluginName

	MaxPollExecuteTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.MaxPollExecuteTimeSecs))
	stats.MaxPollExecuteTime = MaxPollExecuteTime.String()
	AveragePollExecuteTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.AveragePollExecuteTimeSecs))
	stats.AveragePollExecuteTime = AveragePollExecuteTime.String()
	MinPollExecuteTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.MinPollExecuteTimeSecs))
	stats.MinPollExecuteTime = MinPollExecuteTime.String()
	stats.TotalPollQueueLength = pm.Statistics.TotalPollQueueLength
	stats.TotalStandbyPointsLength = pm.Statistics.TotalStandbyPointsLength
	stats.TotalPointsOutForPolling = pm.Statistics.TotalPointsOutForPolling
	stats.ASAPPriorityPollQueueLength = pm.Statistics.ASAPPriorityPollQueueLength
	stats.HighPriorityPollQueueLength = pm.Statistics.HighPriorityPollQueueLength
	stats.NormalPriorityPollQueueLength = pm.Statistics.NormalPriorityPollQueueLength
	stats.LowPriorityPollQueueLength = pm.Statistics.LowPriorityPollQueueLength
	ASAPPriorityAveragePollTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.ASAPPriorityAveragePollTime))
	stats.ASAPPriorityAveragePollTime = ASAPPriorityAveragePollTime.String()
	HighPriorityAveragePollTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.HighPriorityAveragePollTime))
	stats.HighPriorityAveragePollTime = HighPriorityAveragePollTime.String()
	NormalPriorityAveragePollTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.NormalPriorityAveragePollTime))
	stats.NormalPriorityAveragePollTime = NormalPriorityAveragePollTime.String()
	LowPriorityAveragePollTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.LowPriorityAveragePollTime))
	stats.LowPriorityAveragePollTime = LowPriorityAveragePollTime.String()
	stats.TotalPollCount = pm.Statistics.TotalPollCount
	stats.ASAPPriorityPollCount = pm.Statistics.ASAPPriorityPollCount
	stats.HighPriorityPollCount = pm.Statistics.HighPriorityPollCount
	stats.NormalPriorityPollCount = pm.Statistics.NormalPriorityPollCount
	stats.LowPriorityPollCount = pm.Statistics.LowPriorityPollCount
	stats.ASAPPriorityMaxCycleTime = pm.ASAPPriorityMaxCycleTime.String()
	stats.HighPriorityMaxCycleTime = pm.HighPriorityMaxCycleTime.String()
	stats.NormalPriorityMaxCycleTime = pm.NormalPriorityMaxCycleTime.String()
	stats.LowPriorityMaxCycleTime = pm.LowPriorityMaxCycleTime.String()
	stats.ASAPPriorityLockupAlert = pm.Statistics.ASAPPriorityLockupAlert
	stats.HighPriorityLockupAlert = pm.Statistics.HighPriorityLockupAlert
	stats.NormalPriorityLockupAlert = pm.Statistics.NormalPriorityLockupAlert
	stats.LowPriorityLockupAlert = pm.Statistics.LowPriorityLockupAlert
	stats.BusyTime = fmt.Sprintf("%.1f%%", pm.Statistics.BusyTime)
	EnabledTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.EnabledTime))
	stats.EnabledTime = EnabledTime.String()
	PortUnavailableTime, _ := time.ParseDuration(fmt.Sprintf("%fs", pm.Statistics.PortUnavailableTime))
	stats.PortUnavailableTime = PortUnavailableTime.String()

	return &stats
}

func (pm *NetworkPollManager) StartPollingStatistics() {
	pm.pollQueueDebugMsg("StartPollingStatistics()")
	pm.Statistics.PollingStartTimeUnix = time.Now().Unix()
	pm.Statistics.AveragePollExecuteTimeSecs = 0
	pm.Statistics.MaxPollExecuteTimeSecs = 0
	pm.Statistics.MinPollExecuteTimeSecs = 0
	pm.Statistics.ASAPPriorityAveragePollTime = 0
	pm.Statistics.HighPriorityAveragePollTime = 0
	pm.Statistics.NormalPriorityAveragePollTime = 0
	pm.Statistics.LowPriorityAveragePollTime = 0
	pm.Statistics.TotalPollCount = 0
	pm.Statistics.ASAPPriorityPollCount = 0
	pm.Statistics.HighPriorityPollCount = 0
	pm.Statistics.NormalPriorityPollCount = 0
	pm.Statistics.LowPriorityPollCount = 0
	pm.Statistics.ASAPPriorityPollCountForAvg = 0
	pm.Statistics.HighPriorityPollCountForAvg = 0
	pm.Statistics.NormalPriorityPollCountForAvg = 0
	pm.Statistics.LowPriorityPollCountForAvg = 0
	pm.Statistics.ASAPPriorityLockupAlert = false
	pm.Statistics.HighPriorityLockupAlert = false
	pm.Statistics.NormalPriorityLockupAlert = false
	pm.Statistics.LowPriorityLockupAlert = false
	pm.Statistics.PortUnavailableTime = 0
	pm.Statistics.PortUnavailableStartTime = 0
}

func (pm *NetworkPollManager) PollCompleteStatsUpdate(pp *PollingPoint, pollTimeSecs float64) {
	pm.pollQueueDebugMsg("PollCompleteStatsUpdate()")

	if pm.Statistics.MaxPollExecuteTimeSecs == 0 || pollTimeSecs > pm.Statistics.MaxPollExecuteTimeSecs {
		pm.Statistics.MaxPollExecuteTimeSecs = pollTimeSecs
	}
	if pm.Statistics.MinPollExecuteTimeSecs == 0 || pollTimeSecs < pm.Statistics.MinPollExecuteTimeSecs {
		pm.Statistics.MinPollExecuteTimeSecs = pollTimeSecs
	}
	pm.Statistics.AveragePollExecuteTimeSecs = ((pm.Statistics.AveragePollExecuteTimeSecs * float64(pm.Statistics.TotalPollCount)) + pollTimeSecs) / (float64(pm.Statistics.TotalPollCount) + 1)
	pm.Statistics.TotalPollCount++
	pm.Statistics.EnabledTime = time.Since(time.Unix(pm.Statistics.PollingStartTimeUnix, 0)).Seconds()
	pm.Statistics.BusyTime = math.Round((((pm.Statistics.AveragePollExecuteTimeSecs*float64(pm.Statistics.TotalPollCount))/pm.Statistics.EnabledTime)*100)*1000) / 1000 // percentage rounded to 3 decimal places

	pm.Statistics.TotalPollQueueLength = int64(pm.PollQueue.PriorityQueue.Len())
	if pm.PollQueue.QueueUnloader.NextPollPoint != nil {
		pm.Statistics.TotalPollQueueLength++
	}
	pm.Statistics.TotalStandbyPointsLength = int64(pm.PollQueue.StandbyPollingPoints.Len())
	pm.Statistics.TotalPointsOutForPolling = 0
	if pm.PollQueue.QueueUnloader.CurrentPollPoint != nil {
		pm.Statistics.TotalPointsOutForPolling++
	}

	pm.Statistics.ASAPPriorityPollQueueLength = 0
	pm.Statistics.HighPriorityPollQueueLength = 0
	pm.Statistics.NormalPriorityPollQueueLength = 0
	pm.Statistics.LowPriorityPollQueueLength = 0

	for _, pp := range pm.PollQueue.PriorityQueue.priorityQueue {
		if pp != nil {
			switch pp.PollPriority {
			case datatype.PriorityASAP:
				pm.Statistics.ASAPPriorityPollQueueLength++
			case datatype.PriorityHigh:
				pm.Statistics.HighPriorityPollQueueLength++
			case datatype.PriorityNormal:
				pm.Statistics.NormalPriorityPollQueueLength++
			case datatype.PriorityLow:
				pm.Statistics.LowPriorityPollQueueLength++
			}
		}
	}

	switch pp.PollPriority {
	case datatype.PriorityASAP:
		pm.Statistics.ASAPPriorityPollCount++
		if pp.QueueEntryTime <= 0 {
			return
		}
		pollTime := float64(time.Now().Unix() - pp.QueueEntryTime)
		pm.Statistics.ASAPPriorityAveragePollTime = ((pm.Statistics.ASAPPriorityAveragePollTime * float64(pm.Statistics.ASAPPriorityPollCountForAvg)) + pollTime) / (float64(pm.Statistics.ASAPPriorityPollCountForAvg) + 1)
		pm.Statistics.ASAPPriorityPollCountForAvg++

	case datatype.PriorityHigh:
		pm.Statistics.HighPriorityPollCount++
		if pp.QueueEntryTime <= 0 {
			return
		}
		pollTime := float64(time.Now().Unix() - pp.QueueEntryTime)
		pm.Statistics.HighPriorityAveragePollTime = ((pm.Statistics.HighPriorityAveragePollTime * float64(pm.Statistics.HighPriorityPollCountForAvg)) + pollTime) / (float64(pm.Statistics.HighPriorityPollCountForAvg) + 1)
		pm.Statistics.HighPriorityPollCountForAvg++

	case datatype.PriorityNormal:
		pm.Statistics.NormalPriorityPollCount++
		if pp.QueueEntryTime <= 0 {
			return
		}
		pollTime := float64(time.Now().Unix() - pp.QueueEntryTime)
		pm.Statistics.NormalPriorityAveragePollTime = ((pm.Statistics.NormalPriorityAveragePollTime * float64(pm.Statistics.NormalPriorityPollCountForAvg)) + pollTime) / (float64(pm.Statistics.NormalPriorityPollCountForAvg) + 1)
		pm.Statistics.NormalPriorityPollCountForAvg++

	case datatype.PriorityLow:
		pm.Statistics.LowPriorityPollCount++
		if pp.QueueEntryTime <= 0 {
			return
		}
		pollTime := float64(time.Now().Unix() - pp.QueueEntryTime)
		pm.Statistics.LowPriorityAveragePollTime = ((pm.Statistics.LowPriorityAveragePollTime * float64(pm.Statistics.LowPriorityPollCountForAvg)) + pollTime) / (float64(pm.Statistics.LowPriorityPollCountForAvg) + 1)
		pm.Statistics.LowPriorityPollCountForAvg++

	}

}

func (pm *NetworkPollManager) PartialPollStatsUpdate() {
	pm.pollQueueDebugMsg("PartialPollStatsUpdate()")
	pm.Statistics.TotalPollQueueLength = int64(pm.PollQueue.PriorityQueue.Len())
	if pm.PollQueue.QueueUnloader.NextPollPoint != nil {
		pm.Statistics.TotalPollQueueLength++
	}
	pm.Statistics.TotalStandbyPointsLength = int64(pm.PollQueue.StandbyPollingPoints.Len())
	pm.Statistics.TotalPointsOutForPolling = 0
	if pm.PollQueue.QueueUnloader.CurrentPollPoint != nil {
		pm.Statistics.TotalPointsOutForPolling++
	}

	pm.Statistics.EnabledTime = time.Since(time.Unix(pm.Statistics.PollingStartTimeUnix, 0)).Seconds()

	if pm.PortUnavailableTimeout != nil {
		pm.Statistics.PortUnavailableTime += time.Since(time.Unix(pm.Statistics.PortUnavailableStartTime, 0)).Seconds()
		pm.Statistics.PortUnavailableStartTime = time.Now().Unix()
	}

	pm.Statistics.ASAPPriorityPollQueueLength = 0
	pm.Statistics.HighPriorityPollQueueLength = 0
	pm.Statistics.NormalPriorityPollQueueLength = 0
	pm.Statistics.LowPriorityPollQueueLength = 0

	for _, pp := range pm.PollQueue.PriorityQueue.priorityQueue {
		if pp != nil {
			switch pp.PollPriority {
			case datatype.PriorityASAP:
				pm.Statistics.ASAPPriorityPollQueueLength++
			case datatype.PriorityHigh:
				pm.Statistics.HighPriorityPollQueueLength++
			case datatype.PriorityNormal:
				pm.Statistics.NormalPriorityPollQueueLength++
			case datatype.PriorityLow:
				pm.Statistics.LowPriorityPollQueueLength++
			}
		}
	}
}
