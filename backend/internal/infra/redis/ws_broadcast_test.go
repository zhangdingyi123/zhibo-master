package redis

import "testing"

func TestRoomIDFromBroadcastChannel(t *testing.T) {
	tests := []struct {
		channel string
		want    string
	}{
		{"zhibo:room:room_sess_1:broadcast", "room_sess_1"},
		{"zhibo:room:abc:broadcast", "abc"},
		{"other", ""},
	}
	for _, tt := range tests {
		if got := roomIDFromBroadcastChannel(tt.channel); got != tt.want {
			t.Errorf("roomIDFromBroadcastChannel(%q) = %q, want %q", tt.channel, got, tt.want)
		}
	}
}
