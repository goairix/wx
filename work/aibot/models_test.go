package aibot

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

func TestSharedOptionsValidateAndDoNotMutateDependencies(t *testing.T) {
	s, err := settings.Resolve(WithRequestTimeout(time.Second))
	if err != nil || s.RequestTimeout != time.Second || s.MaxDownloadBytes != 20<<20 {
		t.Fatalf("%+v %v", s, err)
	}
	if _, err = settings.Resolve(WithDownloadLimit(0)); err == nil {
		t.Fatal("accepted zero limit")
	}
}
func TestCallbackAllTypesAndRaw(t *testing.T) {
	for _, kind := range []string{"text", "image", "mixed", "voice", "file", "video", "stream", "future"} {
		raw := []byte(`{"msgtype":"` + kind + `","stream":{"id":"s1"},"future_field":123}`)
		cb, err := ParseCallback(raw)
		if err != nil || cb.Message == nil || cb.Message.Stream.ID != "s1" || !bytes.Equal(cb.Raw, raw) {
			t.Fatalf("%+v %v", cb, err)
		}
	}
	cb, err := ParseCallback([]byte(`{"msgtype":"event","event":{"eventtype":"template_card_event","template_card_event":{"task_id":"t1"}}}`))
	if err != nil || cb.Event.Event.TemplateCardEvent.TaskID != "t1" {
		t.Fatalf("%+v %v", cb, err)
	}
}
func TestCardsOfficialFixturesRoundTrip(t *testing.T) {
	for _, kind := range []string{"text_notice", "news_notice", "button_interaction", "vote_interaction", "multiple_interaction"} {
		card := &TemplateCard{CardType: kind, TaskID: "t1", JumpList: []JumpAction{{Type: 3, Question: "next"}}}
		b, err := json.Marshal(card)
		if err != nil || !bytes.Contains(b, []byte(`"question":"next"`)) {
			t.Fatalf("%s %v", b, err)
		}
		if err := ValidateCard(card); err != nil {
			t.Fatal(err)
		}
	}
	raw := RawTemplateCard(`{"card_type":"future","future":false}`)
	if err := ValidateCard(raw); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCard(RawTemplateCard(`[]`)); err == nil {
		t.Fatal("accepted array")
	}
}
