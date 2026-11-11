package whatsmeow_service

import (
	"encoding/json"
	"fmt"

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
