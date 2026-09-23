package jssdk

import "net/url"

func mapValues(key, value string) url.Values {
	return url.Values{key: []string{value}}
}
