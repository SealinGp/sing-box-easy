package fetch

import "testing"

func TestShouldRetryWithClientUA(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{200, false},
		{301, false},
		{401, true}, // some panels answer 401 to an unknown client
		{403, true},
		{404, true},  // the shape seen in the wild: unknown UA -> 404
		{429, false}, // the panel asking to be left alone
		{500, false}, // the panel accepted the request and then failed
		{503, false},
	}
	for _, tt := range tests {
		if got := shouldRetryWithClientUA(tt.status); got != tt.want {
			t.Errorf("shouldRetryWithClientUA(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}
