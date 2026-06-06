package native

import "testing"

func TestToChecksumAddress(t *testing.T) {
	// Official EIP-55 test vectors.
	cases := []string{
		"0x52908400098527886E0F7030069857D2E4169EE7",
		"0x8617E340B3D01FA5F11F306F4090FD50E238070D",
		"0xde709f2102306220921060314715629080e2fb77",
		"0x27b1fdb04752bbc536007a920d24acb045561c26",
		"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		"0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		"0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
		"0xD1220A0cf47c7B9Be7A2E6BA89F429762e7b9aDb",
	}
	for _, want := range cases {
		got := toChecksumAddress(want[2:])
		if got != want {
			t.Errorf("toChecksumAddress(%s) = %s, want %s", want[2:], got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ q, kind, val string }{
		{"0x52908400098527886e0f7030069857d2e4169ee7", "addr", "0x52908400098527886e0f7030069857d2e4169ee7"},
		{"example.eth", "ens", "example.eth"},
		{"eth vitalik.eth", "ens", "vitalik.eth"},
		{"foo.bar.eth", "ens", "foo.bar.eth"},
		{"hello world", "", ""},
		{"0x123", "", ""},
	}
	for _, c := range cases {
		k, v := classify(c.q)
		if k != c.kind || v != c.val {
			t.Errorf("classify(%q) = (%q,%q), want (%q,%q)", c.q, k, v, c.kind, c.val)
		}
	}
}
