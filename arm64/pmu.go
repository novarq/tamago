// ARM64 processor support
// https://github.com/usbarmory/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package arm64

// Performance Monitors register constants
// (ARM Architecture Reference Manual for A-profile architecture)
const (
	// D24.10 PMCR_EL0, Performance Monitors Control Register
	PMCR_LC = 6
	PMCR_C  = 2
	PMCR_E  = 0

	// D24.10 PMCNTENSET_EL0, Performance Monitors Count Enable Set register
	PMCNTENSET_C = 31
)

// defined in pmu.s
func read_pmccntr() uint64
func write_pmcr(val uint32)
func write_pmcntenset(val uint32)

// InitPMU enables the Performance Monitors cycle counter (PMCCNTR_EL0),
// resetting its count.
func (cpu *CPU) InitPMU() {
	write_pmcr(1<<PMCR_LC | 1<<PMCR_C | 1<<PMCR_E)
	write_pmcntenset(1 << PMCNTENSET_C)
}

// Cycles returns the Performance Monitors cycle count (PMCCNTR_EL0), the
// value is meaningful only after [CPU.InitPMU].
func (cpu *CPU) Cycles() uint64 {
	return read_pmccntr()
}
