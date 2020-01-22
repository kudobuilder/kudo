package http

import (
	"testing"
)

func TestIsValidURL(t *testing.T) {

	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{name: "string", uri: "foo", want: false},
		{name: "http", uri: "http://kudo.dev", want: true},
		{name: "https", uri: "https://kudo.dev", want: true},
		{name: "no http prefix", uri: "kudo.dev", want: false},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidURL(tt.uri); got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
