package wallet

import "testing"

func TestGenerateWalletID(t *testing.T) {
	evmAddress := "0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb"

	cases := map[string]string{
		"Ethereum Mainnet": "eth-1Be31A94",
		"BNB Chain":        "bsc-1Be31A94",
		"Arbitrum One":     "arb-1Be31A94",
		"Robinhood Chain":  "rob-1Be31A94",
	}

	for network, want := range cases {
		if got := generateWalletID(network, evmAddress); got != want {
			t.Fatalf("%s: id = %s, mau %s", network, got, want)
		}
	}

	solAddress := "4bhn5Cy15dZxxdrkiH5uwM5PptGkzboYfatQ3v4YUXeV"
	if got := generateWalletID("Solana Mainnet", solAddress); got != "sol-4bhn5Cy1" {
		t.Fatalf("Solana: id = %s, mau sol-4bhn5Cy1", got)
	}
}
