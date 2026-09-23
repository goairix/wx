package webhook

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// Decode unmarshals a parsed payload into a platform event structure.
func Decode(payload Payload, target interface{}) error {
	var err error
	if payload.Format == "json" {
		err = json.Unmarshal(payload.Raw, target)
	} else {
		err = xml.Unmarshal(payload.Raw, target)
	}
	if err != nil {
		return fmt.Errorf("decode webhook event: %w", err)
	}
	return nil
}
