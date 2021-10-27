// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package events

// CollectEvents returns a slice of string consisting
// all the events from the record.FakeRecorder.Events Chan
func CollectEvents(source <-chan string) []string {
	done := false
	events := make([]string, 0)
	for !done {
		select {
		case event := <-source:
			events = append(events, event)
		default:
			done = true
		}
	}
	return events
}

// DrainEvents clears all the events in the chan recorder.Events
// This is a hack as the current byomachine reconciler is global to test
// and the record.FakeRecorder could have events from different tests
// It could also introduce data races in parallel tests run
func DrainEvents(events chan string) {
	for {
		select {
		case <-events:
		default:
			return
		}
	}
}
