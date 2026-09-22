package model

import (
	"encoding/json"
	"testing"
)

func TestCardInfoResponseRelationSupportsNestedCard(t *testing.T) {
	var response CardInfoResponse
	if err := json.Unmarshal([]byte(`{"isSelf":false,"card":{"healthCardId":"hc","relation":"1"}}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Relation != "1" || response.Card.Relation != "1" {
		t.Fatalf("relation=%q card.relation=%q", response.Relation, response.Card.Relation)
	}
}

func TestCardInfoResponseRelationSupportsTopLevel(t *testing.T) {
	var response CardInfoResponse
	if err := json.Unmarshal([]byte(`{"relation":"2","card":{"healthCardId":"hc"}}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Relation != "2" {
		t.Fatalf("relation=%q", response.Relation)
	}
}
