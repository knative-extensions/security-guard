package v1alpha1

import (
	"bytes"
	"fmt"
)

var AsciiFlagNames = []string{
	SpaceSlot:         "Space",
	ExclamationSlot:   "Exclamation",
	DoubleQuoteSlot:   "DoubleQuote",
	NumberSlot:        "NumberSign",
	DollarSlot:        "DollarSign",
	PercentSlot:       "PercentSign",
	SingleQuoteSlot:   "SingleQuote",
	RoundBracketSlot:  "RoundBracket",
	AsteriskSlot:      "MultiplySign",
	PlusSlot:          "PlusSign",
	AtSlot:            "CommentSign",
	MinusSlot:         "MinusSign",
	PeriodSlot:        "DotSign",
	SlashSlot:         "DivideSign",
	ColonSlot:         "ColonSign",
	SemiSlot:          "SemicolonSign",
	LtGtSlot:          "Less/GreaterThanSign",
	EqualSlot:         "EqualSign",
	QuestionSlot:      "QuestionMark",
	CommaSlot:         "CommaSign",
	SquareBracketSlot: "SquareBracket",
	BackslashSlot:     "ReverseDivideSign",
	PowerSlot:         "PowerSign",
	UnderscoreSlot:    "UnderscoreSign",
	AccentSlot:        "AccentSign",
	CurlyBracketSlot:  "CurlyBracket",
	PipeSlot:          "PipeSign",
	NonReadableSlot:   "NonReadableChar",
	CommentsSlot:      "CommentsCombination",
	HexSlot:           "HexCombination",
}

func nameFlags(flags uint32) string {
	var ret bytes.Buffer
	mask := uint32(0x1)

	for i := 0; i < 32; i++ {
		if (flags & mask) != 0 {
			ret.WriteString(AsciiFlagNames[i])
			ret.WriteString(" ")
			flags = flags ^ mask
		}
		mask = mask << 1
	}
	if flags != 0 {
		ret.WriteString("<UnnamedFlags>")
	}
	return ret.String()
}

//////////////////// AsciiFlagsProfile ////////////////

// Exposes ValueProfile interface
type AsciiFlagsProfile uint32

func (profile *AsciiFlagsProfile) profileI(args ...interface{}) {
	profile.Profile(args[0].(uint32))
}

func (profile *AsciiFlagsProfile) Profile(val uint32) {
	*profile = AsciiFlagsProfile(val)
}

//////////////////// AsciiFlagsPile ////////////////

// Exposes ValuePile interface
type AsciiFlagsPile uint32

func (pile *AsciiFlagsPile) addI(valProfile ValueProfile) {
	pile.Add(*valProfile.(*AsciiFlagsProfile))
}

func (pile *AsciiFlagsPile) Add(profile AsciiFlagsProfile) {
	*pile |= AsciiFlagsPile(profile)
}

func (pile *AsciiFlagsPile) Clear() {
	*pile = 0
}

func (pile *AsciiFlagsPile) mergeI(otherValPile ValuePile) {
	pile.Merge(*otherValPile.(*AsciiFlagsPile))
}

func (pile *AsciiFlagsPile) Merge(otherPile AsciiFlagsPile) {
	*pile |= otherPile
}

//////////////////// AsciiFlagsConfig ////////////////

// Exposes ValueConfig interface
type AsciiFlagsConfig uint32

func (config *AsciiFlagsConfig) decideI(valProfile ValueProfile) string {
	return config.Decide(*valProfile.(*AsciiFlagsProfile))
}

func (config *AsciiFlagsConfig) Decide(profile AsciiFlagsProfile) string {
	if flags := AsciiFlagsConfig(profile) & ^*config; flags != 0 {
		return fmt.Sprintf("Unexpected Flags %s (0x%x) in Value", nameFlags(uint32(flags)), flags)
	}
	return ""
}

func (config *AsciiFlagsConfig) learnI(valPile ValuePile) {
	config.Learn(*valPile.(*AsciiFlagsPile))
}

func (config *AsciiFlagsConfig) Learn(pile AsciiFlagsPile) {
	*config = AsciiFlagsConfig(pile)
}

func (config *AsciiFlagsConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(*otherValConfig.(*AsciiFlagsConfig))
}

func (config *AsciiFlagsConfig) Fuse(otherConfig AsciiFlagsConfig) {
	*config |= otherConfig
}
