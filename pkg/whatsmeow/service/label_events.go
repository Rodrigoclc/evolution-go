package whatsmeow_service

import (
	"encoding/json"
	"fmt"
	"reflect"

	label_model "github.com/evolution-foundation/evolution-go/pkg/label/model"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// labelFromEdit converts a LabelEdit event into the stored label and reports whether WhatsApp deleted it.
// Deleted labels are part of full syncs and must be removed instead of upserted, otherwise they would keep
// showing up in /label/list.
func labelFromEdit(instanceID string, evt *events.LabelEdit) (label_model.Label, bool) {
	return label_model.Label{
		InstanceID:   instanceID,
		LabelID:      evt.LabelID,
		LabelName:    evt.Action.GetName(),
		LabelColor:   fmt.Sprintf("%d", evt.Action.GetColor()),
		PredefinedId: fmt.Sprintf("%d", evt.Action.GetPredefinedID()),
	}, evt.Action.GetDeleted()
}

// labelAssociationData returns the webhook payload for a LabelAssociationChat event. When the chat is
// addressed by LID, it adds "JIDAlt" with the phone-number JID (when known), as message events already do
// with SenderAlt/RecipientAlt, so consumers can match the label with the conversation.
func labelAssociationData(evt *events.LabelAssociationChat, resolvePN func(types.JID) (types.JID, error)) interface{} {
	if evt.JID.Server != types.HiddenUserServer || resolvePN == nil {
		return evt
	}
	pn, err := resolvePN(evt.JID.ToNonAD())
	if err != nil || pn.IsEmpty() {
		return evt
	}

	raw, err := json.Marshal(evt)
	if err != nil {
		return evt
	}
	data := map[string]interface{}{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return evt
	}
	data["JIDAlt"] = pn.ToNonAD().String()
	return data
}

// skipFullSyncReplay reports whether rawEvt is a non-label app state event replayed by a full sync
// (FromFullSync=true). Evolution GO only enables full-sync event emission for POST /label/sync, and the
// collection it resyncs may also hold other app state (archive, pin, mute, ...). Those replays are dropped so
// the label resync never notifies subscribers about chat state that did not change. Without the label resync
// whatsmeow emits no FromFullSync events at all, so this keeps the default behavior unchanged.
func skipFullSyncReplay(rawEvt interface{}) bool {
	switch rawEvt.(type) {
	case *events.LabelEdit, *events.LabelAssociationChat, *events.LabelAssociationMessage:
		return false
	}
	v := reflect.ValueOf(rawEvt)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return false
	}
	field := v.FieldByName("FromFullSync")
	return field.IsValid() && field.Kind() == reflect.Bool && field.Bool()
}
