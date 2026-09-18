package response

import "testing"

func TestPreferredResponseFormatUsesQualityValues(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   responseFormat
	}{
		{"higher quality JSON wins when listed second", "application/atom+xml;q=0.8, application/json;odata.metadata=minimal;q=1.0", responseFormatJSON},
		{"zero quality JSON does not win", "application/json;q=0, application/atom+xml;q=1.0", responseFormatAtom},
		{"equal quality retains listed order", "application/atom+xml, application/json", responseFormatAtom},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := preferredResponseFormat(test.accept); got != test.want {
				t.Fatalf("preferredResponseFormat(%q) = %v, want %v", test.accept, got, test.want)
			}
		})
	}
}
