package v1alpha1

import (
	"fmt"
	"math/bits"
	"net"
)

//////////////////// IpSetProfile ////////////////

// Exposes ValueProfile interface
type IpSetProfile []net.IP

func (profile *IpSetProfile) profileI(args ...interface{}) {
	switch v := args[0].(type) {
	case string:
		profile.ProfileString(v)
	case net.IP:
		profile.ProfileIP(v)
	case []net.IP:
		profile.ProfileIPSlice(v)
	default:
		panic("Unsupported type in IpSetProfile")
	}
}

func (profile *IpSetProfile) ProfileString(str string) {
	*profile = nil
	if len(str) > 0 {
		if ip := net.ParseIP(str); ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsPrivate() {
			if ipv4 := ip.To4(); ipv4 != nil {
				ip = ipv4
			}
			*profile = append(*profile, ip)
		}
	}
}

func (profile *IpSetProfile) ProfileIP(ip net.IP) {
	*profile = nil
	if ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsPrivate() {
		*profile = IpSetProfile{ip}
	}
}

func (profile *IpSetProfile) ProfileIPSlice(ipSlice []net.IP) {
	*profile = nil
	for _, ip := range ipSlice {

		if ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsPrivate() {
			dup := make(net.IP, len(ip))
			copy(dup, ip)
			*profile = append(*profile, dup)
		}
	}
}

//////////////////// IpSetPile ////////////////

// Exposes ValuePile interface
// During json.Marshal(), IpSetPile exposes only the List
// After json.Unmarshal(), the map will be nil even when the List is not empty
// If the map is nil, it should be populated from the information in List
// If the map is populated it is always kept in-sync with the information in List
type IpSetPile struct {
	List []net.IP
	m    map[string]bool
}

func (pile *IpSetPile) addI(valProfile ValueProfile) {
	pile.Add(valProfile.(*IpSetProfile))
}

func (pile *IpSetPile) Add(profile *IpSetProfile) {
	if pile.m == nil {
		pile.m = make(map[string]bool, len(pile.List)+16)
		// Populate the map from the information in List
		for _, v := range pile.List {
			pile.m[v.String()] = true
		}
	}
	for _, v := range *profile {
		ipStr := v.String()
		if !pile.m[ipStr] {
			pile.m[ipStr] = true
			pile.List = append(pile.List, v)
		}
	}
}

func (pile *IpSetPile) Clear() {
	pile.m = nil
	pile.List = nil
}

func (pile *IpSetPile) mergeI(otherValPile ValuePile) {
	pile.Merge(otherValPile.(*IpSetPile))
}

func (pile *IpSetPile) Merge(otherPile *IpSetPile) {
	if pile.List == nil {
		pile.List = otherPile.List
		pile.m = otherPile.m
		return
	}

	if pile.m == nil {
		pile.m = make(map[string]bool, len(pile.List)+len(otherPile.List))
		// Populate the map from the information in List
		for _, v := range pile.List {
			pile.m[v.String()] = true
		}
	}
	for _, v := range otherPile.List {
		ipStr := v.String()
		if !pile.m[ipStr] {
			pile.m[ipStr] = true
			pile.List = append(pile.List, v)
		}
	}
}

//////////////////// CidrSetConfig ////////////////

type CIDR net.IPNet

func (cidr *CIDR) lastIP() net.IP {
	var lastIp net.IP = make(net.IP, len(cidr.IP))
	copy(lastIp, cidr.IP)
	for i, b := range cidr.Mask {
		lastIp[i] |= ^b
	}
	return lastIp
}

// Return true if cidr include the ip range of otherCidr
func (cidr *CIDR) Include(otherCidr CIDR) bool {
	// Check first IP of otherCidr
	if (*net.IPNet)(cidr).Contains(otherCidr.IP) {
		// Check last IP of otherCidr
		return (*net.IPNet)(cidr).Contains(otherCidr.lastIP())
	}
	return false
}

// InflateBy try to add IP to a CIDR by extending the CIDR mask
// The maximal extension allowed by the implementation is a C Subnet
// (i.e. mask of 255.255.255.0 in IPv4)
// InflateBy returns true if successful
func (cidr *CIDR) InflateBy(ip net.IP) bool {
	if (*net.IPNet)(cidr).Contains(ip) {
		return true
	}

	cidrLen := len(cidr.IP)
	if len(ip) != cidrLen {
		// never try to mingle ipv4 and ipv6 staff
		return false
	}

	// lets try to inflate
	cidrLast := cidrLen - 1
	cidrBits := cidrLen * 8
	// Is the Ip in the same C Subnet as the CIDR?
	for i := 0; i < cidrLast; i++ {
		xor := cidr.IP[i] ^ ip[i]
		if xor != 0 {
			// Avoid creating cidrs larger than C Subnets
			return false
		}
	}

	// Inflate the CIDR to cover the IP as well!
	xor := cidr.IP[cidrLast] ^ ip[cidrLast]
	bitsShared := bits.LeadingZeros8(xor)
	mask := net.CIDRMask(cidrBits-8+bitsShared, cidrBits)
	cidr.IP = cidr.IP.Mask(mask)
	cidr.Mask[cidrLast] &= mask[cidrLast]
	return true
}

// Exposes ValueConfig interface
type IpSetConfig []CIDR

func (config *IpSetConfig) decideI(valProfile ValueProfile) string {
	return config.Decide((valProfile.(*IpSetProfile)))
}

func (config *IpSetConfig) Decide(profile *IpSetProfile) string {
	if len(*profile) == 0 {
		return ""
	}

LoopProfileIPs:
	for _, ip := range *profile {
		if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() {
			continue LoopProfileIPs
		}
		for _, subnet := range *config {
			if (*net.IPNet)(&subnet).Contains(ip) {
				continue LoopProfileIPs
			}
		}
		return fmt.Sprintf("IP %s not allowed", ip.String())
	}
	return ""
}

// Learn currently offers a rough and simple CIDR support
// Learn try to add IPs to current CIDRs by inflating the CIDRs.
// When no CIDR can be inflated to include the IP, Learn adds a new CIDR for this IP
func (config *IpSetConfig) learnI(valPile ValuePile) {
	config.Learn(valPile.(*IpSetPile))
}

func (config *IpSetConfig) Learn(pile *IpSetPile) {
	*config = nil
LoopPileIPs:
	for _, ip := range pile.List {
		for _, cidr := range *config {
			if cidr.InflateBy(ip) {
				continue LoopPileIPs
			}
		}
		// Unsuccessful inflating CIDRs to include IP
		if len(ip) == 4 {
			*config = append(*config, (CIDR)(net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}))
		} else {
			*config = append(*config, (CIDR)(net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}))
		}
	}
}

// Fuse CidrSetConfig
// The implementation look to opportunistically skip new entries
// The implementation does not squash new and old entries
// The implementation does not fuse to squash consecutive cidrs
func (config *IpSetConfig) fuseI(otherValConfig ValueConfig) {
	config.Fuse(otherValConfig.(*IpSetConfig))
}

func (config *IpSetConfig) Fuse(otherConfig *IpSetConfig) {
LoopOtherCidrs:
	for _, otherCidr := range *otherConfig {
		for idx, myCidr := range *config {
			if myCidr.InflateBy(otherCidr.IP) {
				if myCidr.InflateBy(otherCidr.lastIP()) {
					continue LoopOtherCidrs
				}
			}
			if myCidr.Include(otherCidr) {
				continue LoopOtherCidrs
			}
			if otherCidr.Include(myCidr) {
				(*config)[idx] = otherCidr
				continue LoopOtherCidrs
			}
		}
		// Add the otherCidr to my list of CIDRs
		*config = append(*config, otherCidr)
	}
}
