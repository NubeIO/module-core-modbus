package pollqueue

import (
	"fmt"

	"github.com/NubeIO/lib-utils-go/nstring"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
)

func (pm *NetworkPollManager) pollQueueDebugMsg(args ...interface{}) {
	if nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "DEBUG") {
		prefix := fmt.Sprintf("%s Poll Queue: ", pm.PluginName)
		log.Info(prefix, args)
	}
}

func (pm *NetworkPollManager) pollQueuePollingMsg(args ...interface{}) {
	if nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "POLLING") || nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "DEBUG") {
		prefix := fmt.Sprintf("%s Poll Queue: ", pm.PluginName)
		log.Info(prefix, args)
	}
}

func (pm *NetworkPollManager) pollQueueErrorMsg(args ...interface{}) {
	prefix := fmt.Sprintf("%s Poll Queue: ", pm.PluginName)
	log.Error(prefix, args)
}

func (pm *NetworkPollManager) PrintPollQueuePointUUIDs() {
	if nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "DEBUG") { // Added here to disable debug processes when not using logging
		printString := "\n\n"
		printString += fmt.Sprint("NextPollPoint: ")
		printString += fmt.Sprint("PollQueue: COUNT = ", pm.PollQueue.PriorityQueue.Len(), ": ")
		for _, pp := range pm.PollQueue.PriorityQueue.priorityQueue {
			printString += fmt.Sprint(pp.FFPointUUID, " - ", pp.PollPriority, "; ")
			if pp != nil {
				printString += fmt.Sprint(pp.FFPointUUID, " - ", pp.PollPriority, "; ")
			} else {
				pm.pollQueueErrorMsg("PrintPollQueuePointUUIDs() for _, pp := range pm.PollQueue.PriorityQueue.PriorityQueue: pp is nil")
			}
		}
		printString += fmt.Sprint("", "\n")
		printString += fmt.Sprint("StandbyPollingPoints COUNT = ", pm.PollQueue.StandbyPollingPoints.Len(), ": ")
		for _, pp := range pm.PollQueue.StandbyPollingPoints.queue {
			if pp != nil {
				printString += fmt.Sprint(pp.FFPointUUID, " - ", pp.PollPriority, ", repoll:", pp.RepollTimer != nil, "; ")
			} else {
				pm.pollQueueErrorMsg("PrintPollQueuePointUUIDs() for _, pp := range pm.PollQueue.StandbyPollingPoints.PriorityQueue: pp is nil")
			}
		}
		printString += fmt.Sprint("\n")
		pm.pollQueueDebugMsg(printString)
	}
}

func (pm *NetworkPollManager) PrintPointDebugInfo(pnt *model.Point) {
	if nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "DEBUG") { // Added here to disable debug processes when not using logging
		printString := "\n\n"
		if pnt != nil {
			printString += fmt.Sprint("Point: ", pnt.UUID, " ", pnt.Name, "\n")
			printString += fmt.Sprint("WriteMode: ", pnt.WriteMode, "\n")
			if pnt.WritePollRequired != nil {
				printString += fmt.Sprint("WritePollRequired: ", *pnt.WritePollRequired, "\n")
			}
			if pnt.ReadPollRequired != nil {
				printString += fmt.Sprint("ReadPollRequired: ", *pnt.ReadPollRequired, "\n")
			}
			if pnt.WriteValue == nil {
				printString += fmt.Sprint("WriteValue: nil", "\n")
			} else {
				printString += fmt.Sprint("WriteValue: ", *pnt.WriteValue, "\n")
			}
			if pnt.OriginalValue == nil {
				printString += fmt.Sprint("OriginalValue: nil", "\n")
			} else {
				printString += fmt.Sprint("OriginalValue: ", *pnt.OriginalValue, "\n")
			}
			if pnt.PresentValue == nil {
				printString += fmt.Sprint("PresentValue: nil", "\n")
			} else {
				printString += fmt.Sprint("PresentValue: ", *pnt.PresentValue, "\n")
			}
			if pnt.CurrentPriority == nil {
				printString += fmt.Sprint("CurrentPriority: nil", "\n")
			} else {
				printString += fmt.Sprint("CurrentPriority: ", *pnt.CurrentPriority, "\n")
			}
			if pnt.Priority != nil {
				if pnt.Priority.P1 != nil {
					printString += fmt.Sprint("_1: ", *pnt.Priority.P1, "\n")
				}
				if pnt.Priority.P2 != nil {
					printString += fmt.Sprint("_2: ", *pnt.Priority.P2, "\n")
				}
				if pnt.Priority.P3 != nil {
					printString += fmt.Sprint("_3: ", *pnt.Priority.P3, "\n")
				}
				if pnt.Priority.P4 != nil {
					printString += fmt.Sprint("_4: ", *pnt.Priority.P4, "\n")
				}
				if pnt.Priority.P5 != nil {
					printString += fmt.Sprint("_5: ", *pnt.Priority.P5, "\n")
				}
				if pnt.Priority.P6 != nil {
					printString += fmt.Sprint("_6: ", *pnt.Priority.P6, "\n")
				}
				if pnt.Priority.P7 != nil {
					printString += fmt.Sprint("_7: ", *pnt.Priority.P7, "\n")
				}
				if pnt.Priority.P8 != nil {
					printString += fmt.Sprint("_8: ", *pnt.Priority.P8, "\n")
				}
				if pnt.Priority.P9 != nil {
					printString += fmt.Sprint("_9: ", *pnt.Priority.P9, "\n")
				}
				if pnt.Priority.P10 != nil {
					printString += fmt.Sprint("_10: ", *pnt.Priority.P10, "\n")
				}
				if pnt.Priority.P11 != nil {
					printString += fmt.Sprint("_11: ", *pnt.Priority.P11, "\n")
				}
				if pnt.Priority.P12 != nil {
					printString += fmt.Sprint("_12: ", *pnt.Priority.P12, "\n")
				}
				if pnt.Priority.P13 != nil {
					printString += fmt.Sprint("_13: ", *pnt.Priority.P13, "\n")
				}
				if pnt.Priority.P14 != nil {
					printString += fmt.Sprint("_14: ", *pnt.Priority.P14, "\n")
				}
				if pnt.Priority.P15 != nil {
					printString += fmt.Sprint("_15: ", *pnt.Priority.P15, "\n")
				}
				if pnt.Priority.P16 != nil {
					printString += fmt.Sprint("_16: ", *pnt.Priority.P16, "\n")
				}
			}
			pm.pollQueueDebugMsg(printString)
			return
		}
		pm.pollQueueDebugMsg("ERROR: INVALID POINT")
	}
}

func (pm *NetworkPollManager) PrintPollingPointDebugInfo(pp *PollingPoint) {
	if pp != nil {
		pm.pollQueueDebugMsg(fmt.Sprintf("PollingPoint pp %+v", pp))
	}
}

func (pm *NetworkPollManager) PrintPollQueueStatistics() {
	if nstring.IsEqualIgnoreCase(pm.Config.LogLevel, "DEBUG") { // Added here to disable debug processes when not using logging

		pm.PrintPollQueuePointUUIDs()

		printString := "\n\n"
		printString += fmt.Sprint("PrintPollQueueStatistics: \n")
		printString += fmt.Sprint("TotalPollQueueLength: ", pm.PollQueue.PriorityQueue.Len(), "\n")
		printString += fmt.Sprint("TotalStandbyPointsLength: ", pm.PollQueue.StandbyPollingPoints.Len(), "\n")
		printString += fmt.Sprint("ASAPPriorityMaxCycleTime: ", pm.ASAPPriorityMaxCycleTime, "\n")
		printString += fmt.Sprint("HighPriorityMaxCycleTime: ", pm.HighPriorityMaxCycleTime, "\n")
		printString += fmt.Sprint("NormalPriorityMaxCycleTime: ", pm.NormalPriorityMaxCycleTime, "\n")
		printString += fmt.Sprint("LowPriorityMaxCycleTime: ", pm.LowPriorityMaxCycleTime, "\n")
		printString += fmt.Sprint("MaxPollExecuteTimeSecs: ", pm.Statistics.MaxPollExecuteTimeSecs, "\n")
		printString += fmt.Sprint("AveragePollExecuteTimeSecs: ", pm.Statistics.AveragePollExecuteTimeSecs, "\n")
		printString += fmt.Sprint("MinPollExecuteTimeSecs: ", pm.Statistics.MinPollExecuteTimeSecs, "\n")
		printString += fmt.Sprint("ASAPPriorityPollQueueLength: ", pm.Statistics.ASAPPriorityPollQueueLength, "\n")
		printString += fmt.Sprint("HighPriorityPollQueueLength: ", pm.Statistics.HighPriorityPollQueueLength, "\n")
		printString += fmt.Sprint("NormalPriorityPollQueueLength: ", pm.Statistics.NormalPriorityPollQueueLength, "\n")
		printString += fmt.Sprint("LowPriorityPollQueueLength: ", pm.Statistics.LowPriorityPollQueueLength, "\n")
		printString += fmt.Sprint("ASAPPriorityAveragePollTime: ", pm.Statistics.ASAPPriorityAveragePollTime, "\n")
		printString += fmt.Sprint("HighPriorityAveragePollTime: ", pm.Statistics.HighPriorityAveragePollTime, "\n")
		printString += fmt.Sprint("NormalPriorityAveragePollTime: ", pm.Statistics.NormalPriorityAveragePollTime, "\n")
		printString += fmt.Sprint("LowPriorityAveragePollTime: ", pm.Statistics.LowPriorityAveragePollTime, "\n")
		printString += fmt.Sprint("TotalPollCount: ", pm.Statistics.TotalPollCount, "\n")
		printString += fmt.Sprint("ASAPPriorityPollCount: ", pm.Statistics.ASAPPriorityPollCount, "\n")
		printString += fmt.Sprint("HighPriorityPollCount: ", pm.Statistics.HighPriorityPollCount, "\n")
		printString += fmt.Sprint("NormalPriorityPollCount: ", pm.Statistics.NormalPriorityPollCount, "\n")
		printString += fmt.Sprint("LowPriorityPollCount: ", pm.Statistics.LowPriorityPollCount, "\n")
		printString += fmt.Sprint("ASAPPriorityPollCountForAvg: ", pm.Statistics.ASAPPriorityPollCountForAvg, "\n")
		printString += fmt.Sprint("HighPriorityPollCountForAvg: ", pm.Statistics.HighPriorityPollCountForAvg, "\n")
		printString += fmt.Sprint("NormalPriorityPollCountForAvg: ", pm.Statistics.NormalPriorityPollCountForAvg, "\n")
		printString += fmt.Sprint("LowPriorityPollCountForAvg: ", pm.Statistics.LowPriorityPollCountForAvg, "\n")
		printString += fmt.Sprint("ASAPPriorityLockupAlert: ", pm.Statistics.ASAPPriorityLockupAlert, "\n")
		printString += fmt.Sprint("HighPriorityLockupAlert: ", pm.Statistics.HighPriorityLockupAlert, "\n")
		printString += fmt.Sprint("NormalPriorityLockupAlert: ", pm.Statistics.NormalPriorityLockupAlert, "\n")
		printString += fmt.Sprint("LowPriorityLockupAlert: ", pm.Statistics.LowPriorityLockupAlert, "\n")
		printString += fmt.Sprint("PollingStartTimeUnix: ", pm.Statistics.PollingStartTimeUnix, "\n")
		printString += fmt.Sprint("BusyTime: ", pm.Statistics.BusyTime, "% \n")
		printString += fmt.Sprint("EnabledTime: ", pm.Statistics.EnabledTime, " \n")
		printString += fmt.Sprint("PortUnavailableTime: ", pm.Statistics.PortUnavailableTime, " \n")
		printString += fmt.Sprint("\n")
		pm.pollQueueDebugMsg(printString)
	}
}
