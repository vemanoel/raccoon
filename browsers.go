package cookies

import "strings"

const (
	Firefox    = "firefox"
	FirefoxDev = "firefox-dev"
	Zen        = "zen"
	Tor        = "tor"
	Floorp     = "floorp"
	Waterfox   = "waterfox"
	LibreWolf  = "librewolf"
	Mullvad    = "mullvad"
	Midori     = "midori"
	SeaMonkey  = "sea-monkey"
	GNUIceCat  = "gnu-ice-cat"
	KMeleon    = "k-meleon"
	PaleMoon   = "pale-moon"
	Basilisk   = "basilisk"
)

func SupportedBrowsers() string {
	elems := []string{
		Firefox,
		FirefoxDev,
		Zen,
		Tor,
		Floorp,
		Waterfox,
		LibreWolf,
		Mullvad,
		Midori,
		SeaMonkey,
		GNUIceCat,
		KMeleon,
		PaleMoon,
		Basilisk,
	}
	return strings.Join(elems, " ,")
}
