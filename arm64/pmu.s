// ARM64 processor support
// https://github.com/usbarmory/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

#include "arm64.h"

// func read_pmccntr() uint64
TEXT ·read_pmccntr(SB),$0-8
	// ARM Architecture Reference Manual for A-profile architecture
	// D24.10 PMCCNTR_EL0, Performance Monitors Cycle Count Register
	ISB	SY
	MRS	PMCCNTR_EL0, R0
	MOVD	R0, ret+0(FP)

	RET

// func write_pmcr(val uint32)
TEXT ·write_pmcr(SB),$0-4
	// ARM Architecture Reference Manual for A-profile architecture
	// D24.10 PMCR_EL0, Performance Monitors Control Register
	MOVW	val+0(FP), R0
	MSR	R0, PMCR_EL0
	ISB	SY

	RET

// func write_pmcntenset(val uint32)
TEXT ·write_pmcntenset(SB),$0-4
	// ARM Architecture Reference Manual for A-profile architecture
	// D24.10 PMCNTENSET_EL0, Performance Monitors Count Enable Set register
	MOVW	val+0(FP), R0
	MSR	R0, PMCNTENSET_EL0
	ISB	SY

	RET
