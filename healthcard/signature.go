package healthcard

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func sign(values map[string]interface{}, appSecret string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "sign" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if value == nil {
			continue
		}

		var rendered string
		if stringValue, ok := value.(string); ok {
			rendered = stringValue
		} else {
			data, err := json.Marshal(value)
			if err != nil {
				rendered = fmt.Sprint(value)
			} else {
				rendered = string(data)
			}
		}
		if rendered == "" {
			continue
		}
		parts = append(parts, key+"="+rendered)
	}

	digest := sha256.Sum256([]byte(strings.Join(parts, "&") + appSecret))
	return base64.StdEncoding.EncodeToString(digest[:])
}
