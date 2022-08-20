package v1alpha1

import (
	"fmt"
	"net"
	"testing"
)

func TestIpSet_IPSLICE(t *testing.T) {
	arguments := [][][]net.IP{
		{{{1, 2, 3, 4}}},
		{{{1, 2, 3, 5}}},
		{{{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4}}},
		{{{2, 2, 2, 2}}},
		{{{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 5}}},
		{{{2, 2, 2, 4}}},
		{{{4, 2, 2, 2}}},
		{{{2, 0, 0, 2}}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(IpSetProfile))
		piles = append(piles, new(IpSetPile))
		configs = append(configs, new(IpSetConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestIpSet_IP(t *testing.T) {
	arguments := [][]net.IP{
		{{1, 2, 3, 4}},
		{{1, 2, 3, 5}},
		{{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4}},
		{{2, 2, 2, 2}},
		{{1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 5}},
		{{2, 2, 2, 4}},
		{{4, 2, 2, 2}},
		{{2, 0, 0, 2}},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(IpSetProfile))
		piles = append(piles, new(IpSetPile))
		configs = append(configs, new(IpSetConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}

func TestIpSet_STRING(t *testing.T) {
	arguments := [][]string{
		{"1.2.3.4"},
		{"1.2.3.5"},
		{"ff02::1:ff07"},
		{"2.2.2.2"},
		{"ff02::7:ff07"},
		{"2.2.2.4"},
		{"ff02::1:ff09"},
		{"2.0.0.2"},
	}
	var args []interface{}
	var profiles []ValueProfile
	var piles []ValuePile
	var configs []ValueConfig
	for i := 0; i < 10; i++ {
		profiles = append(profiles, new(IpSetProfile))
		piles = append(piles, new(IpSetPile))
		configs = append(configs, new(IpSetConfig))
	}
	for i := 0; i < len(arguments); i++ {
		args = append(args, arguments[i])
	}
	ValueTests_All(t, profiles, piles, configs, args...)
}
func TestIpSet_Config(t *testing.T) {
	t.Run("CIDR management", func(t *testing.T) {
		IpSet1first := new(IpSetProfile)
		IpSet1second := new(IpSetProfile)
		IpSet1Test := new(IpSetProfile)
		IpSet1Loopback := new(IpSetProfile)
		IpSet1Private := new(IpSetProfile)
		IpSet1Unspecified := new(IpSetProfile)
		IpSet2first := new(IpSetProfile)
		IpSet2second := new(IpSetProfile)
		IpSet2Test := new(IpSetProfile)
		IpSet2Bad := new(IpSetProfile)
		IpSet6first := new(IpSetProfile)
		IpSet6second := new(IpSetProfile)
		IpSet6Test := new(IpSetProfile)
		IpSet6Bad := new(IpSetProfile)
		IpSet6Loopback := new(IpSetProfile)
		IpSet6Private := new(IpSetProfile)
		IpSet6Unspecified := new(IpSetProfile)

		fmt.Println(net.ParseIP("111.7.1.126"))
		IpSet1first.Profile([]net.IP{net.ParseIP("111.7.1.126")})
		IpSet1second.Profile([]net.IP{net.ParseIP("111.7.1.129")})
		IpSet1Test.Profile([]net.IP{net.ParseIP("111.7.1.70")})
		IpSet1Loopback.Profile([]net.IP{net.ParseIP("127.1.5.70")})
		IpSet1Private.Profile([]net.IP{net.ParseIP("10.17.33.70")})
		IpSet1Unspecified.Profile([]net.IP{net.ParseIP("0.0.0.0")})
		IpSet2first.Profile([]net.IP{net.ParseIP("111.7.2.10")})
		IpSet2second.Profile([]net.IP{net.ParseIP("111.7.2.20")})
		IpSet2Test.Profile([]net.IP{net.ParseIP("111.7.2.15")})
		IpSet2Bad.Profile([]net.IP{net.ParseIP("111.7.2.200")})
		IpSet6first.Profile([]net.IP{net.ParseIP("ff02::1:ff07")})
		IpSet6second.Profile([]net.IP{net.ParseIP("ff02::1:ff09")})
		IpSet6Test.Profile([]net.IP{net.ParseIP("ff02::1:ff08")})
		IpSet6Bad.Profile([]net.IP{net.ParseIP("ff02::2:ff08")})
		IpSet6Loopback.Profile([]net.IP{net.ParseIP("::1")})
		IpSet6Private.Profile([]net.IP{net.ParseIP("fc00::3")})
		IpSet6Unspecified.Profile([]net.IP{net.ParseIP("::")})

		// A case where a new IP inflate the CIDR to a C subnet
		pile1 := new(IpSetPile)
		config1 := new(IpSetConfig)
		pile1.Add(IpSet1first)
		pile1.Add(IpSet1second)
		config1.Learn(pile1)

		if ret := config1.Decide(IpSet1first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config1.Decide(IpSet1second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config1.Decide(IpSet1Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config1.Decide(IpSet2Test); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
		if ret := config1.Decide(IpSet1Loopback); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config1.Decide(IpSet1Private); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config1.Decide(IpSet1Unspecified); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}

		// A case where a new IP inflate the CIDR to smaller subnet
		pile2 := new(IpSetPile)
		config2 := new(IpSetConfig)
		pile2.Add(IpSet2first)
		pile2.Add(IpSet2second)
		config2.Learn(pile2)

		if ret := config2.Decide(IpSet2first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config2.Decide(IpSet2second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config2.Decide(IpSet2Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config2.Decide(IpSet2Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
		if ret := config2.Decide(IpSet1Test); ret == "" {
			t.Errorf("Expected ip to fail!")
		}

		// A case for two subnets
		pile3 := new(IpSetPile)
		config3 := new(IpSetConfig)
		pile3.Add(IpSet1first)
		pile3.Add(IpSet1second)
		pile3.Add(IpSet2first)
		pile3.Add(IpSet2second)
		config3.Learn(pile3)

		if ret := config3.Decide(IpSet1first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet1second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet1Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet2first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet2second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet2Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config3.Decide(IpSet2Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}

		// A case where a new IP does not need to inflate the CIDR
		pile4 := new(IpSetPile)
		config4 := new(IpSetConfig)
		pile4.Add(IpSet1first)
		pile4.Add(IpSet1second)
		pile4.Add(IpSet1Test)
		config4.Learn(pile4)
		if ret := config4.Decide(IpSet1Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config4.Decide(IpSet2Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}

		// A case for ipv6
		pile6 := new(IpSetPile)
		config6 := new(IpSetConfig)
		pile6.Add(IpSet6first)
		pile6.Add(IpSet6second)
		config6.Learn(pile6)

		if ret := config6.Decide(IpSet6first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config6.Decide(IpSet6second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config6.Decide(IpSet6Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config6.Decide(IpSet6Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
		if ret := config6.Decide(IpSet2Test); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
		if ret := config6.Decide(IpSet6Loopback); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config6.Decide(IpSet6Private); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config6.Decide(IpSet6Unspecified); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}

		// A case of ipv4 and ipv6 together
		pile7 := new(IpSetPile)
		config7 := new(IpSetConfig)
		pile7.Add(IpSet1first)
		pile7.Add(IpSet1second)
		pile7.Add(IpSet2first)
		pile7.Add(IpSet2second)
		pile7.Add(IpSet6first)
		pile7.Add(IpSet6second)
		config7.Learn(pile7)
		if ret := config7.Decide(IpSet6first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet6second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet6Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet6Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
		if ret := config7.Decide(IpSet1first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet1second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet1Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet2first); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet2second); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet2Test); ret != "" {
			t.Errorf("Expected ip to be accepted but received %s", ret)
		}
		if ret := config7.Decide(IpSet2Bad); ret == "" {
			t.Errorf("Expected ip to fail!")
		}
	})

}
