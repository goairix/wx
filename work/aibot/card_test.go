package aibot

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Fixtures are the five official examples in https://developer.work.weixin.qq.com/document/path/101032.
func TestCardsCompleteOfficialFixtures(t *testing.T) {
	for _, kind := range []string{"text_notice", "news_notice", "button_interaction", "vote_interaction", "multiple_interaction"} {
		t.Run(kind, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + kind + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var card TemplateCard
			if err = json.Unmarshal(data, &card); err != nil {
				t.Fatal(err)
			}
			wire, err := json.Marshal(&card)
			if err != nil {
				t.Fatal(err)
			}
			var want, got any
			_ = json.Unmarshal(data, &want)
			_ = json.Unmarshal(wire, &got)
			compareJSON(t, want, got)
		})
	}
}
func compareJSON(t *testing.T, want, got any) {
	t.Helper()
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			t.Fatal("object lost")
		}
		for k, v := range w {
			gv, exists := g[k]
			if !exists {
				if v == "" || v == float64(0) {
					continue
				}
				t.Fatalf("field %s lost", k)
			}
			compareJSON(t, v, gv)
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			t.Fatal("array lost")
		}
		for i, v := range w {
			compareJSON(t, v, g[i])
		}
	default:
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("%v != %v", want, got)
		}
	}
}

func TestTypedCardPreservesFeedback(t *testing.T) {
	card := &TemplateCard{CardType: "text_notice", Feedback: &Feedback{ID: "feedback"}}
	b, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["feedback"].(map[string]any)["id"] != "feedback" {
		t.Fatal("feedback lost")
	}
}
