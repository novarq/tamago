// Microchip Two-wire Interface support
// https://github.com/usbarmory/tamago
//
// Copyright (c) The TamaGo Authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package twi implements the Two-wire Interface controller found in Microchip
// FLEXCOM peripherals under the following specification:
//   - Microchip - LAN9694/LAN9696/LAN9698 Datasheet - DS00005048E (02-27-25)
//
// This package is only meant to be used with `GOOS=tamago GOARCH=arm64` as
// supported by the TamaGo framework for bare metal Go, see
// https://github.com/usbarmory/tamago.
package twi

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/usbarmory/tamago/bits"
	"github.com/usbarmory/tamago/internal/reg"
)

const (
	flexMode          = 0x000
	modeOperation     = 0
	modeOperationMask = 0x3
	modeTWI           = 3

	twiControl       = 0x600
	controlStart     = 0
	controlStop      = 1
	controlMasterENA = 2
	controlReset     = 7

	twiMasterMode              = 0x604
	masterModeInternalSize     = 8
	masterModeInternalSizeMask = 0x3
	masterModeRead             = 12
	masterModeAddress          = 16
	masterModeAddressMask      = 0x7f

	twiInternalAddress = 0x60c

	twiClock           = 0x610
	clockLowDivider    = 0
	clockLowMask       = 0xff
	clockHighDivider   = 8
	clockHighMask      = 0xff
	clockCommonDivider = 16
	clockCommonMask    = 0x7
	clockGeneric       = 20

	twiStatus            = 0x620
	statusTransferDone   = 0
	statusReceiveReady   = 1
	statusTransmitReady  = 2
	statusNotAcknowledge = 8

	twiReceive  = 0x630
	twiTransmit = 0x634

	maximumInternalAddressSize = 3
)

const defaultTimeout = 100 * time.Millisecond

// TWI represents a Microchip FLEXCOM controller in TWI initiator mode.
type TWI struct {
	sync.Mutex

	// Base register of the FLEXCOM peripheral.
	Base uint32
	// ClockLowDivider configures the low-period divider.
	ClockLowDivider uint8
	// ClockHighDivider configures the high-period divider.
	ClockHighDivider uint8
	// ClockDivider applies a 2^n scale to both clock periods.
	ClockDivider uint8
	// GenericClock selects GCLK instead of the peripheral clock.
	GenericClock bool
	// Timeout bounds each wait for controller progress. Zero selects 100 ms.
	Timeout time.Duration

	clock uint32
}

// Init configures and enables TWI initiator mode. Generic clock generation and
// pin routing are configured separately by the SoC or board package.
func (hw *TWI) Init() (err error) {
	switch {
	case hw.Base == 0:
		return errors.New("invalid TWI controller instance")
	case hw.ClockDivider > clockCommonMask:
		return fmt.Errorf("invalid TWI clock divider %d", hw.ClockDivider)
	case hw.Timeout < 0:
		return fmt.Errorf("invalid TWI timeout %s", hw.Timeout)
	}

	if hw.Timeout == 0 {
		hw.Timeout = defaultTimeout
	}

	bits.SetN(&hw.clock, clockLowDivider, clockLowMask, uint32(hw.ClockLowDivider))
	bits.SetN(&hw.clock, clockHighDivider, clockHighMask, uint32(hw.ClockHighDivider))
	bits.SetN(&hw.clock, clockCommonDivider, clockCommonMask, uint32(hw.ClockDivider))
	bits.SetTo(&hw.clock, clockGeneric, hw.GenericClock)

	hw.Lock()
	defer hw.Unlock()

	reg.SetN(hw.Base+flexMode, modeOperation, modeOperationMask, modeTWI)
	hw.reset()

	return
}

func (hw *TWI) reset() {
	reg.Write(hw.Base+twiControl, 1<<controlReset)
	reg.Write(hw.Base+twiClock, hw.clock)
	reg.Write(hw.Base+twiControl, 1<<controlMasterENA)
}

func (hw *TWI) configure(address uint8, internal uint32, internalSize int, read bool) (err error) {
	switch {
	case hw.Base == 0:
		return errors.New("invalid TWI controller instance")
	case address > masterModeAddressMask:
		return fmt.Errorf("invalid I2C address %#x", address)
	case internalSize < 0 || internalSize > maximumInternalAddressSize:
		return fmt.Errorf("invalid I2C internal address size %d", internalSize)
	case internalSize == 0 && internal != 0:
		return fmt.Errorf("invalid I2C internal address %#x", internal)
	case internalSize > 0 && internal >= 1<<uint(internalSize*8):
		return fmt.Errorf("invalid I2C internal address %#x", internal)
	}

	var mode uint32
	bits.SetN(&mode, masterModeInternalSize, masterModeInternalSizeMask, uint32(internalSize))
	bits.SetTo(&mode, masterModeRead, read)
	bits.SetN(&mode, masterModeAddress, masterModeAddressMask, uint32(address))
	reg.Write(hw.Base+twiMasterMode, mode)
	reg.Write(hw.Base+twiInternalAddress, internal)

	return
}

func (hw *TWI) wait(bit int) (err error) {
	start := time.Now()

	for {
		status := reg.Read(hw.Base + twiStatus)

		switch {
		case status&(1<<statusNotAcknowledge) != 0:
			hw.reset()
			return errors.New("i2c target did not acknowledge")
		case status&(1<<uint(bit)) != 0:
			return nil
		case time.Since(start) >= hw.Timeout:
			hw.reset()
			return fmt.Errorf("i2c timeout waiting for status bit %d (%#08x)", bit, status)
		}

		runtime.Gosched()
	}
}

// Read receives bytes from a 7-bit I2C target. internalSize selects zero to
// three address bytes; internal must fit the selected width.
func (hw *TWI) Read(address uint8, internal uint32, internalSize int, buf []byte) (err error) {
	if len(buf) == 0 {
		return
	}

	hw.Lock()
	defer hw.Unlock()

	if err = hw.configure(address, internal, internalSize, true); err != nil {
		return
	}

	reg.Read(hw.Base + twiStatus)

	if len(buf) == 1 {
		reg.Write(hw.Base+twiControl, 1<<controlStart|1<<controlStop)
		if err = hw.wait(statusReceiveReady); err != nil {
			return
		}
		buf[0] = byte(reg.Read(hw.Base + twiReceive))
	} else {
		reg.Write(hw.Base+twiControl, 1<<controlStart)
		for i := range buf {
			if i == len(buf)-1 {
				reg.Write(hw.Base+twiControl, 1<<controlStop)
			}
			if err = hw.wait(statusReceiveReady); err != nil {
				return
			}
			buf[i] = byte(reg.Read(hw.Base + twiReceive))
		}
	}

	err = hw.wait(statusTransferDone)

	return
}

// Write sends bytes to a 7-bit I2C target. internalSize selects zero to three
// address bytes; internal must fit the selected width.
func (hw *TWI) Write(address uint8, internal uint32, internalSize int, buf []byte) (err error) {
	if len(buf) == 0 {
		return
	}

	hw.Lock()
	defer hw.Unlock()

	if err = hw.configure(address, internal, internalSize, false); err != nil {
		return
	}

	reg.Read(hw.Base + twiStatus)

	for i, value := range buf {
		reg.Write(hw.Base+twiTransmit, uint32(value))
		if i == len(buf)-1 {
			reg.Write(hw.Base+twiControl, 1<<controlStop)
		}
		if err = hw.wait(statusTransmitReady); err != nil {
			return
		}
	}

	err = hw.wait(statusTransferDone)

	return
}
