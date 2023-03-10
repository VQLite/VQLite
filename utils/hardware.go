package utils

import (
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"

	"runtime"
)

func GetAvailableMemory() uint64 {
	stats, err := mem.VirtualMemory()
	if err != nil {
		log.Error().Err(err).Msg("get free memory size error")
		return 0
	}
	return stats.Available
}

func GetCpuCount() int {
	count := runtime.NumCPU()
	return count
}

func GetCpuUsage() float64 {

	percents, err := cpu.Percent(0, false)

	if err != nil {
		log.Error().Err(err).Msg("get cpu usage error")
		return 0
	}

	return percents[0]
}
