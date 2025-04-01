// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

// VMState defines the VM state
type VMState int

const (
	// VMStateUnknown indicates the VM state unknown
	VMStateUnknown VMState = iota
	// VMStateStart indicates the VM state start
	VMStateStart
	// VMStateStop indicates the VM state stop
	VMStateStop
	// VMStateRestart indicates the VM state restart
	VMStateRestart
)

// String returns the VM state in string format
func (v VMState) String() string {
	return [...]string{"Unknown", "Start", "Stop", "Restart"}[v]
}
