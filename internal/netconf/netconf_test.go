package netconf

import "testing"

func TestNetmask(t *testing.T) {
	cases := map[int]string{8: "255.0.0.0", 16: "255.255.0.0", 24: "255.255.255.0", 32: "255.255.255.255"}
	for bits, want := range cases {
		if got := Netmask(bits); got != want {
			t.Errorf("Netmask(%d) = %s, want %s", bits, got, want)
		}
	}
}
