// ARM64 processor support
// https://github.com/usbarmory/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package arm64

import (
	"math"
)

// TicklessIdleGovernor is an optional CPU idle time management function which
// halts the CPU until the next scheduler deadline, accounting the halted time
// for [CPU.IdleTime].
//
// The governor wakes through the physical timer interrupt (see
// [CPU.SetAlarm]), the application must enable and service TIMER_IRQ (see
// [ServiceInterrupts]) before setting the governor, otherwise the CPU halts
// past its deadline.
func (cpu *CPU) TicklessIdleGovernor(pollUntil int64) {
	if pollUntil == 0 {
		return
	}

	if pollUntil != math.MaxInt64 {
		cpu.SetAlarm(pollUntil)
	}

	t0 := read_cntpct()
	cpu.WaitInterrupt()
	cpu.idleTicks.Add(int64(read_cntpct() - t0))
}

// IdleTime returns the cumulative time in nanoseconds spent halted by
// [CPU.TicklessIdleGovernor].
func (cpu *CPU) IdleTime() int64 {
	return int64(float64(cpu.idleTicks.Load()) * cpu.TimerMultiplier)
}
