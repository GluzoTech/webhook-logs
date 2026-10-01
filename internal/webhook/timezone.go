package webhook

import (
	"time"
	// Embed the zone database so the IST lookup below works on hosts that ship
	// without one, such as Windows and scratch containers.
	_ "time/tzdata"
)

// istLocation is the timezone every timestamp is recorded, bucketed and
// rendered in, so a day's log file holds exactly that Indian calendar day no
// matter where the server runs.
var istLocation = loadIST()

func loadIST() *time.Location {
	if loc, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		return loc
	}
	// UTC+05:30, which India has observed without change since 1945.
	return time.FixedZone("IST", 5*60*60+30*60)
}
