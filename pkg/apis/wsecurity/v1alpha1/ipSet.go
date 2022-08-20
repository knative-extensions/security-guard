package v1alpha1

import (
	"fmt"
	"math/bits"
	"net"
)

//////////////////// IpSetProfile ////////////////

// Exposes ValueProfile interface
// A Slice of IPs
type IpSetProfile []net.IP

func (profile *IpSetProfile) Profile(args ...interface{}) {
	switch v := args[0].(type) {
	case string:
		if len(v) > 0 {
			if ip := net.ParseIP(v); ip != nil {
				*profile = append(*profile, ip)
			}
		}

	case net.IP:
		if v != nil {
			*profile = append(*profile, v)
		}
	case []net.IP:
		*profile = nil
		for _, ip := range v {
			if ip != nil {
				dup := make(net.IP, len(ip))
				copy(dup, ip)
				*profile = append(*profile, dup)
			}
		}
	default:
		panic("Unsupported type in IpSetProfile")
	}
}

//////////////////// IpSetPile ////////////////

// Exposes ValuePile interface
// During json.Marsjal(), IpSetPile exposes only the List
// After json.Unmarshal(), the map will be nil even when the List is not empty
// If the map is nil, it should be populated from the information in List
// If the map is populated it is always kept in-sync with the information in List
type IpSetPile struct {
	List []net.IP
	m    map[string]bool
}

func (pile *IpSetPile) Add(valProfile ValueProfile) {
	profile := valProfile.(*IpSetProfile)

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

func (pile *IpSetPile) Merge(otherValPile ValuePile) {
	otherPile := otherValPile.(*IpSetPile)

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

// Return true if cidr include the ip range of otherCidr
func (cidr *CIDR) Include(otherCidr CIDR) bool {
	// Check first IP of otherCidr
	if (*net.IPNet)(cidr).Contains(otherCidr.IP) {
		// Check last IP of otherCidr
		var lastIp net.IP = make(net.IP, len(otherCidr.IP))
		copy(lastIp, otherCidr.IP)
		fmt.Println("lastIp", len(lastIp))
		fmt.Println("otherCidr.IP", len(otherCidr.IP))
		fmt.Println("otherCidr.Mask", len(otherCidr.Mask))
		for i, b := range otherCidr.Mask {
			lastIp[i] |= ^b
		}
		return (*net.IPNet)(cidr).Contains(lastIp)
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

func (config *IpSetConfig) Decide(valProfile ValueProfile) string {
	profile := (valProfile.(*IpSetProfile))

	for _, cidr := range *config {
		fmt.Println("Start Decide config ", (net.IPNet(cidr)).IP.String(), (net.IPNet(cidr)).Mask.String())
	}
	for _, ip := range *profile {
		fmt.Println("Start Decide profile ", ip.String())
	}
	fmt.Println("Start Decide", *config, *profile)
	if len(*profile) == 0 {
		fmt.Println("len(*profile) == 0")
		return ""
	}

LoopProfileIPs:
	for _, ip := range *profile {
		fmt.Println("IP in Decide", ip)
		if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() {
			fmt.Println("ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ", ip)
			continue LoopProfileIPs
		}
		for _, subnet := range *config {
			fmt.Println("IP in Decide", ip, "Subnet", subnet)
			if (*net.IPNet)(&subnet).Contains(ip) {
				fmt.Println("(*net.IPNet)(&subnet).Contains(ip)")
				continue LoopProfileIPs
			}
		}
		fmt.Println("IP not allowed")
		return fmt.Sprintf("IP %s not allowed", ip.String())
	}
	fmt.Println("Decide done")

	return ""
}

// Learn currently offers a rough and simple CIDR support
// Learn try to add IPs to current CIDRs by inflating the CIDRs.
// When no CIDR can be inflated to include the IP, Learn adds a new CIDR for this IP
func (config *IpSetConfig) Learn(valPile ValuePile) {
	pile := valPile.(*IpSetPile)

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
// Future: Improve Fuse to squash consecutive cidrs
func (config *IpSetConfig) Fuse(otherValConfig ValueConfig) {
	otherConfig := otherValConfig.(*IpSetConfig)

LoopOtherCidrs:
	for _, otherCidr := range *otherConfig {
		for idx, myCidr := range *config {
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
