package cli

import "testing"

func TestDeriveScreenName(t *testing.T) {
	cases := map[string]string{
		"HomeActivity":           "Home",
		"SettingsActivity":       "Settings",
		"CheckoutViewController": "Checkout",
		"home":                   "Home",
		"settings":               "Settings",
		"user_profile":           "User Profile",
		"order-history":          "Order History",
		"promo/details":          "Promo Details",
		"PromoSheet":             "Promo Sheet",
	}
	for in, want := range cases {
		if got := deriveScreenName(in); got != want {
			t.Errorf("deriveScreenName(%q) = %q, want %q", in, got, want)
		}
	}
}
