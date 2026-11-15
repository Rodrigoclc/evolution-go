package label_service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	label_model "github.com/evolution-foundation/evolution-go/pkg/label/model"
	"go.mau.fi/whatsmeow/appstate"
)

// labelSyncTimeout bounds the full resync of the "regular" app state collection.
const labelSyncTimeout = 2 * time.Minute

// forcedResyncLocks serializes forced resyncs per instance (key: instance id -> *sync.Mutex): the event
// emission option is a client field and must not be toggled by concurrent calls for the same instance, while
// different instances must not wait on each other.
var forcedResyncLocks sync.Map

// runWithFullSyncEvents enables event emission for full syncs only while run executes and restores the
// previous value afterwards, on success or error.
//
// whatsmeow does not emit app state events (LabelEdit, LabelAssociationChat, ...) during a full sync, such
// as the one right after QR pairing, unless EmitAppStateEventsOnFullSync is set. Without those events the
// label table stays empty and chats that were already labeled never reach the webhook. Enabling the option
// only here keeps the default instance behavior unchanged.
func runWithFullSyncEvents(key string, flag *bool, run func() error) error {
	lock, _ := forcedResyncLocks.LoadOrStore(key, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	previous := *flag
	*flag = true
	defer func() { *flag = previous }()
	return run()
}

// SyncLabels runs a full resync of the "regular" app state collection, where WhatsApp keeps labels and
// label-chat associations, emitting the matching events. The event handlers refresh the label table
// (LabelEdit) and deliver LabelEdit/LabelAssociationChat to the instance webhook for LABEL subscribers.
// It returns the current labels of the instance.
func (l *labelService) SyncLabels(instance *instance_model.Instance) ([]label_model.Label, error) {
	client, err := l.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	l.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Starting full label resync (app state %s)", instance.Id, appstate.WAPatchRegular)
	err = runWithFullSyncEvents(instance.Id, &client.EmitAppStateEventsOnFullSync, func() error {
		// The timeout starts only after the per-instance lock is held.
		ctx, cancel := context.WithTimeout(context.Background(), labelSyncTimeout)
		defer cancel()
		return client.FetchAppState(ctx, appstate.WAPatchRegular, true, false)
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("label resync timed out after %s: %w", labelSyncTimeout, err)
		}
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Full label resync failed: %v", instance.Id, err)
		return nil, err
	}

	labels, err := l.labelRepository.GetAllLabelsByInstanceID(instance.Id)
	if err != nil {
		l.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error fetching labels from database: %v", instance.Id, err)
		return nil, err
	}
	l.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Full label resync completed: %d label(s)", instance.Id, len(labels))
	return labels, nil
}
