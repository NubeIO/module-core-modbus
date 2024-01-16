package pollqueue

import (
	"fmt"
	"time"

	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-utils-go/boolean"
	"github.com/NubeIO/lib-utils-go/float"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
)

// REFS:
//  - GOLANG HEAP https://pkg.go.dev/container/heap
//  - Worker Queue tutorial: https://www.opsdash.com/blog/job-queues-in-go.html

type Config struct {
	EnablePolling bool   `yaml:"enable_polling"`
	LogLevel      string `yaml:"log_level"`
}

type NetworkPollManager struct {
	Config     *Config
	Marshaller nmodule.Marshaller

	Enable                    bool
	PollQueue                 *NetworkPriorityPollQueue
	StatsCalcTimer            *time.Ticker
	PortUnavailableTimeout    *time.Timer
	QueueCheckerTimer         *time.Ticker
	QueueCheckerCancelChannel chan bool
	DeviceDurations           map[string][]time.Duration

	// References
	FFNetworkUUID string
	NetworkName   string
	FFPluginUUID  string
	PluginName    string

	// Settings
	ASAPPriorityMaxCycleTime   time.Duration // threshold setting for triggering a lockup alert for ASAP priority.
	HighPriorityMaxCycleTime   time.Duration // threshold setting for triggering a lockup alert for High priority.
	NormalPriorityMaxCycleTime time.Duration // threshold setting for triggering a lockup alert for Normal priority.
	LowPriorityMaxCycleTime    time.Duration // threshold setting for triggering a lockup alert for Low priority.

	// Stats
	Statistics PollStatistics
}

func (pm *NetworkPollManager) StartPolling() {
	pm.SetAllDevicePollRateDurations()
	pm.RebuildPollingQueue()
	pm.Enable = true
	pm.PollQueue.Start()
	pm.StartQueueCheckerAndStats()
	pm.StartPollingStatistics()
}

func (pm *NetworkPollManager) StopPolling() {
	pm.Enable = false
	pm.PollQueue.Stop()
	if pm.QueueCheckerTimer != nil && pm.QueueCheckerCancelChannel != nil {
		pm.StopQueueCheckerAndStats()
	}
}

func (pm *NetworkPollManager) PausePolling() {
	pm.pollQueueDebugMsg("PausePolling()")
	pm.Enable = false
}

func (pm *NetworkPollManager) UnpausePolling() {
	pm.pollQueueDebugMsg("UnpausePolling()")
	pm.Enable = true
	pm.PortUnavailableTimeout = nil
}

func (pm *NetworkPollManager) ReAddDevicePoints(devUUID string) { // This is triggered by a user who wants to update the device poll times for standby points
	dev, err := pm.Marshaller.GetDevice(devUUID, &nmodule.Opts{Args: &nargs.Args{WithPoints: true}})
	if dev == nil || err != nil {
		pm.pollQueueErrorMsg("ReAddDevicePoints(): cannot find device ", devUUID)
		return
	}
	pm.PollQueue.RemovePollingPointByDeviceUUID(devUUID)
	for _, pnt := range dev.Points {
		if boolean.IsTrue(pnt.Enable) {
			pp := NewPollingPoint(pnt.UUID, pnt.DeviceUUID, dev.NetworkUUID)
			pp.PollPriority = pnt.PollPriority
			pm.PollQueue.AddToPriorityQueue(pp)
		}
	}
}

func NewPollManager(conf *Config, marshaller nmodule.Marshaller, ffNetworkUUID, ffNetworkName, pluginName string) *NetworkPollManager {
	pm := new(NetworkPollManager)
	pm.Enable = false
	pm.Config = conf
	pm.PollQueue = NewNetworkPriorityPollQueue(conf)
	pm.Marshaller = marshaller
	pm.FFNetworkUUID = ffNetworkUUID
	pm.NetworkName = ffNetworkName
	pm.PluginName = pluginName
	pm.ASAPPriorityMaxCycleTime, _ = time.ParseDuration("2m")
	pm.HighPriorityMaxCycleTime, _ = time.ParseDuration("5m")
	pm.NormalPriorityMaxCycleTime, _ = time.ParseDuration("15m")
	pm.LowPriorityMaxCycleTime, _ = time.ParseDuration("60m")
	return pm
}

func (pm *NetworkPollManager) SetAllDevicePollRateDurations() {
	net, _ := pm.Marshaller.GetNetwork(pm.FFNetworkUUID, &nmodule.Opts{Args: &nargs.Args{WithDevices: true}})
	pm.DeviceDurations = make(map[string][]time.Duration, len(net.Devices))
	for _, dev := range net.Devices {
		pm.SetDevicePollRateDurations(dev)
	}
}

func (pm *NetworkPollManager) SetDevicePollRateDurations(device *model.Device) {
	defFast := 10 * time.Second
	defNorm := 30 * time.Second
	defSlow := 120 * time.Second

	fastRateDuration, _ := time.ParseDuration(fmt.Sprintf("%fs", float.NonNil(device.FastPollRate)))
	if fastRateDuration <= 100*time.Millisecond {
		fastRateDuration = defFast
	}

	normalRateDuration, _ := time.ParseDuration(fmt.Sprintf("%fs", float.NonNil(device.NormalPollRate)))
	if normalRateDuration <= 500*time.Millisecond {
		normalRateDuration = defNorm
	}

	slowRateDuration, _ := time.ParseDuration(fmt.Sprintf("%fs", float.NonNil(device.SlowPollRate)))
	if slowRateDuration <= 1*time.Second {
		slowRateDuration = defSlow
	}

	pm.DeviceDurations[device.UUID] = []time.Duration{fastRateDuration, normalRateDuration, slowRateDuration}
}

func (pm *NetworkPollManager) GetPollRateDuration(rate datatype.PollRate, deviceUUID string) time.Duration {
	switch rate {
	case datatype.RateFast:
		return pm.DeviceDurations[deviceUUID][0]
	case datatype.RateNormal:
		return pm.DeviceDurations[deviceUUID][1]
	case datatype.RateSlow:
		return pm.DeviceDurations[deviceUUID][2]
	default:
		pm.pollQueueDebugMsg("GetPollRateDuration(): UNKNOWN", deviceUUID)
		return pm.DeviceDurations[deviceUUID][2]
	}
}

func (pm *NetworkPollManager) SinglePollFinished(pp *PollingPoint, point *model.Point, pollStartTime time.Time, writeSuccess, readSuccess, pollingWasNotRequired bool, retryType PollRetryType) {
	pollEndTime := time.Now()
	pollDuration := pollEndTime.Sub(pollStartTime)
	pollTimeSecs := pollDuration.Seconds()
	pm.PollingPointCompleteNotification(pp, point, writeSuccess, readSuccess, pollTimeSecs, false, true, retryType, pollingWasNotRequired)
}

func (pm *NetworkPollManager) PollQueueErrorChecking() {
	pm.pollQueueDebugMsg("pollQueue error check")
	net, err := pm.Marshaller.GetNetwork(pm.FFNetworkUUID, &nmodule.Opts{Args: &nargs.Args{WithDevices: true, WithPoints: true}})
	if err != nil {
		pm.pollQueueErrorMsg("pollQueue error check: Network Not Found")
		return
	}
	if boolean.IsFalse(net.Enable) {
		if pm.PollQueue.PriorityQueue.Len() > 0 {
			pm.pollQueueErrorMsg("pollQueue error check: Found PollingPoints in PriorityQueue of a disabled network")
			pm.PollQueue.PriorityQueue.EmptyQueue()
		}
		if pm.PollQueue.StandbyPollingPoints.Len() > 0 {
			pm.pollQueueErrorMsg("pollQueue error check: Found PollingPoints in StandbyPollingPoints of a disabled network")
			pm.PollQueue.StandbyPollingPoints.EmptyQueue()
		}
	}
	for _, dev := range net.Devices {
		for _, pnt := range dev.Points {
			pp := pm.PollQueue.GetPollingPointByPointUUID(pnt.UUID)
			if boolean.IsFalse(dev.Enable) {
				if pp != nil {
					pm.pollQueueErrorMsg("pollQueue error check: Found point in poll queue of disabled device", pnt.Name, pnt.UUID)
					pm.PollQueue.RemovePollingPointByDeviceUUID(dev.UUID)
				}
				continue
			}
			if boolean.IsFalse(pnt.Enable) {
				if pp != nil {
					pm.pollQueueErrorMsg("pollQueue error check: Found disabled point in poll queue ", pnt.Name, pnt.UUID)
					pm.PollQueue.RemovePollingPointByPointUUID(pnt.UUID)
				}
				continue
			}
			if pp == nil {
				pm.pollQueueErrorMsg("pollQueue error check: Polling point doesn't exist for point ", pnt.Name, pnt.UUID)
				pp = NewPollingPoint(pnt.UUID, pnt.DeviceUUID, dev.NetworkUUID)
				pm.PollingPointCompleteNotification(pp, pnt, false, false, 0, true, true, NORMAL_RETRY, false) // This will perform the queue re-add actions based on Point WriteMode.
				continue
			}
		}
	}
}

func (pm *NetworkPollManager) StartQueueCheckerAndStats() {
	if pm.QueueCheckerTimer != nil {
		pm.QueueCheckerTimer.Stop()
	}
	if pm.QueueCheckerCancelChannel != nil {
		pm.QueueCheckerCancelChannel <- true
	}

	pm.QueueCheckerTimer = time.NewTicker(5 * time.Minute)
	pm.QueueCheckerCancelChannel = make(chan bool)
	go func() {
		for {
			select {
			case <-pm.QueueCheckerCancelChannel:
				return
			case <-pm.QueueCheckerTimer.C:
				pm.PollQueueErrorChecking()
				pm.PrintPollQueueStatistics()
			}
		}
	}()
}

func (pm *NetworkPollManager) StopQueueCheckerAndStats() {
	pm.QueueCheckerTimer.Stop()
	pm.QueueCheckerTimer = nil
	pm.QueueCheckerCancelChannel <- true
	pm.QueueCheckerCancelChannel = nil
}

func (pm *NetworkPollManager) PortUnavailable() {
	pm.Statistics.PortUnavailableStartTime = time.Now().Unix()
	pm.PausePolling()
}

func (pm *NetworkPollManager) PortAvailable() {
	pm.PartialPollStatsUpdate()
	pm.PrintPollQueueStatistics()
	pm.UnpausePolling()
}

func PollOnStartCheck(pnt *model.Point) bool {
	return pnt.PollOnStartup == nil || boolean.IsTrue(pnt.PollOnStartup)
}
