package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SNetwork struct {
	wire *SWire

	AvailableIpAddressCount *int `json:"availableIpAddressCount,omitempty"`
	ID                      string
	Name                    string
	Properties              SubnetPropertiesFormat
	AddressPrefix           string `json:"addressPrefix,omitempty"`
}

func (self *SNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SNetwork) GetId() string {
	return self.ID
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SNetwork) IsEmulated() bool {
	return false
}

func (self *SNetwork) GetStatus() string {
	return "available"
}

func (self *SNetwork) Delete() error {
	vpc := self.wire.vpc
	subnets := []SNetwork{}
	if vpc.Properties.Subnets != nil {
		for i := 0; i < len(*vpc.Properties.Subnets); i++ {
			if (*vpc.Properties.Subnets)[i].Name == self.Name {
				continue
			}
			subnets = append(subnets, (*vpc.Properties.Subnets)[i])
		}
		vpc.Properties.Subnets = &subnets
		vpc.Properties.ProvisioningState = ""
		return self.wire.vpc.region.client.Update(jsonutils.Marshal(vpc), nil)
	}
	return nil
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	return pref.MaskLen
}

// https://docs.microsoft.com/en-us/azure/virtual-network/virtual-networks-faq
func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.Properties.AddressPrefix)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	startIp = startIp.StepUp()                    // 3
	startIp = startIp.StepUp()                    // 4
	return startIp.String()
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) Refresh() error {
	if new, err := self.wire.zone.region.GetNetworkDetail(self.ID); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) GetProjectId() string {
	return getResourceGroup(self.ID)
}