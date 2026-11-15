package whatsmeow_service

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestLabelFromEditMapsLabelAndDeletion(t *testing.T) {
	evt := &events.LabelEdit{
		LabelID: "3",
		Action: &waSyncAction.LabelEditAction{
			Name:         proto.String("Aguardando crédito"),
			Color:        proto.Int32(5),
			PredefinedID: proto.Int32(0),
		},
	}

	label, deleted := labelFromEdit("instance-1", evt)

	if deleted {
		t.Fatal("label must not be reported as deleted")
	}
	if label.InstanceID != "instance-1" || label.LabelID != "3" || label.LabelName != "Aguardando crédito" || label.LabelColor != "5" {
		t.Fatalf("unexpected label: %+v", label)
	}

	evt.Action.Deleted = proto.Bool(true)
	if _, deleted = labelFromEdit("instance-1", evt); !deleted {
		t.Fatal("deleted label must be reported as deleted")
	}
}

func TestLabelFromEditToleratesMissingAction(t *testing.T) {
	label, deleted := labelFromEdit("instance-1", &events.LabelEdit{LabelID: "9"})

	if deleted || label.LabelID != "9" || label.LabelName != "" {
		t.Fatalf("unexpected result for an event without action: %+v deleted=%v", label, deleted)
	}
}

func TestLabelAssociationDataAddsPhoneForLID(t *testing.T) {
	lid := types.NewJID("123456789012345", types.HiddenUserServer)
	pn := types.NewJID("5561993214567", types.DefaultUserServer)
	evt := &events.LabelAssociationChat{
		JID:       lid,
		Timestamp: time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
		LabelID:   "3",
		Action:    &waSyncAction.LabelAssociationAction{Labeled: proto.Bool(true)},
	}

	data := labelAssociationData(evt, func(j types.JID) (types.JID, error) {
		if j != lid {
			t.Fatalf("resolver called with %s", j)
		}
		return pn, nil
	})

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["JIDAlt"] != "5561993214567@s.whatsapp.net" {
		t.Fatalf("JIDAlt missing or wrong: %v", payload["JIDAlt"])
	}
	if payload["JID"] != "123456789012345@lid" || payload["LabelID"] != "3" {
		t.Fatalf("original fields must be kept: %v", payload)
	}
	action, ok := payload["Action"].(map[string]interface{})
	if !ok || action["labeled"] != true {
		t.Fatalf("label action must be kept: %v", payload["Action"])
	}
}

func TestLabelAssociationDataKeepsEventWhenNoPhoneIsNeededOrKnown(t *testing.T) {
	pnChat := &events.LabelAssociationChat{JID: types.NewJID("5561993214567", types.DefaultUserServer), LabelID: "3"}
	called := false
	if got := labelAssociationData(pnChat, func(types.JID) (types.JID, error) { called = true; return types.EmptyJID, nil }); got != pnChat || called {
		t.Fatal("phone-number chats must be sent unchanged without lookups")
	}

	lidChat := &events.LabelAssociationChat{JID: types.NewJID("123456789012345", types.HiddenUserServer), LabelID: "3"}
	if got := labelAssociationData(lidChat, func(types.JID) (types.JID, error) { return types.EmptyJID, nil }); got != lidChat {
		t.Fatal("unknown LID mapping must keep the original event")
	}
	if got := labelAssociationData(lidChat, func(types.JID) (types.JID, error) { return types.EmptyJID, errors.New("store") }); got != lidChat {
		t.Fatal("lookup errors must keep the original event")
	}
	if got := labelAssociationData(lidChat, nil); got != lidChat {
		t.Fatal("missing resolver must keep the original event")
	}
}

func TestSkipFullSyncReplayKeepsOnlyLabelEvents(t *testing.T) {
	cases := []struct {
		name string
		evt  interface{}
		skip bool
	}{
		{"archive replayed by full sync", &events.Archive{FromFullSync: true}, true},
		{"pin replayed by full sync", &events.Pin{FromFullSync: true}, true},
		{"archive changed now", &events.Archive{FromFullSync: false}, false},
		{"label edit from full sync", &events.LabelEdit{FromFullSync: true}, false},
		{"label association from full sync", &events.LabelAssociationChat{FromFullSync: true}, false},
		{"label on message from full sync", &events.LabelAssociationMessage{FromFullSync: true}, false},
		{"event without FromFullSync", &events.Connected{}, false},
		{"nil event", nil, false},
		{"typed nil pointer", (*events.Archive)(nil), false},
	}
	for _, c := range cases {
		if got := skipFullSyncReplay(c.evt); got != c.skip {
			t.Errorf("%s: skip=%v, want %v", c.name, got, c.skip)
		}
	}
}
