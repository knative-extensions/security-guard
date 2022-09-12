package v1alpha1

type Ctrl struct {
	Alert bool `json:"alert"` // If true, use SessionDataConfig to identify alerts
	Block bool `json:"block"` // If true, block on alert.
	Learn bool `json:"learn"` // If true, and no alert identified, report piles
	Force bool `json:"force"` // If true, learning is done even when alert identified, report piles
	Auto  bool `json:"auto"`  // If true, use learned SessionDataConfig rather than configured SessionDataConfig
}

type GuardianSpec struct {
	Configured *SessionDataConfig `json:"configured"`        // configrued criteria
	Learned    *SessionDataConfig `json:"learned,omitempty"` // Learned citeria
	Control    *Ctrl              `json:"control"`           // Control
}

// AutoActivate is a Guardian operation mode that is useful for security automation of new services
func (g *GuardianSpec) AutoActivate() {
	if g.Control == nil {
		g.Control = new(Ctrl)
	}
	g.Control.Auto = true
	g.Control.Learn = true
	g.Control.Force = true
	g.Control.Alert = true
}
