package metaso

import (
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/pin"
)

type PevHandle struct {
	BlockInfoData *MetaSoBlockInfo
	HostMap map[string]*MetaSoBlockNDV
	AddressMap map[string]*MetaSoBlockMDV
	HostAddressMap map[string]*MetaSoHostAddress
}
func HandlePevSlice(pinList []pin.PinInscription,data *PevHandle, block *MetaBlockChainData,blockHeight int64,blockTime int64,isMempool bool) (err error) {
	var pevList []PEVData
	allowProtocols := common.Config.Statistics.AllowProtocols
	allowHost := common.Config.Statistics.AllowHost

	for _, pinNode := range pinList {
		if pinNode.Host == "metabitcoin.unknown" {
			continue
		}
		if pinNode.Host == "" {
			pinNode.Host = "metabitcoin.unknown"
		}
		if len(allowProtocols) >= 1 && allowProtocols[0] != "*" {
			if !ArrayExist(pinNode.Path, allowProtocols) {
				continue
			}
		}
		if len(allowHost) >= 1 && allowHost[0] != "*" {
			if !ArrayExist(pinNode.Host, allowHost) {
				continue
			}
		}
		pevs, err := pb.CountPDV(blockHeight, block, &pinNode)
		if err != nil {
			continue
		}
		for _, pev := range pevs {
			if pev.ToPINId == "" {
				continue
			}
			if pev.Host == "" || len(pev.Host) == 0 {
				pev.Host = "metabitcoin.unknown"
			}
			pevList = append(pevList, pev)
		}
	}
	

	hostMap := make(map[string]struct{})
	addressMap := make(map[string]struct{})
	for _, pev := range pevList {
		hostMap[pev.Host] = struct{}{}
		addressMap[pev.Address] = struct{}{}
		data.BlockInfoData.DataValue = data.BlockInfoData.DataValue.Add(pev.IncrementalValue)
		data.BlockInfoData.PinNumber += 1
		if pev.Host != "metabitcoin.unknown" {
			data.BlockInfoData.PinNumberHasHost += 1
		}
		//handle host and address
		if _, ok := data.HostMap[pev.Host]; ok {
			data.HostMap[pev.Host].DataValue = data.HostMap[pev.Host].DataValue.Add(pev.IncrementalValue)
			data.HostMap[pev.Host].PinNumber += 1
		} else {
			data.HostMap[pev.Host] = &MetaSoBlockNDV{DataValue: pev.IncrementalValue, Block: blockHeight, Host: pev.Host, PinNumber: 1, BlockTime: blockTime}
		}
		if _, ok := data.AddressMap[pev.Address]; ok {
			data.AddressMap[pev.Address].DataValue = data.AddressMap[pev.Address].DataValue.Add(pev.IncrementalValue)
			data.AddressMap[pev.Address].PinNumber += 1
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			data.AddressMap[pev.Address].PinNumberHasHost += t
		} else {
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			data.AddressMap[pev.Address] = &MetaSoBlockMDV{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime}
		}
		hostAddress := fmt.Sprintf("%s--%s", pev.Host, pev.Address)
		if _, ok := data.HostAddressMap[hostAddress]; ok {
			data.HostAddressMap[hostAddress].DataValue = data.HostAddressMap[hostAddress].DataValue.Add(pev.IncrementalValue)
			data.HostAddressMap[hostAddress].PinNumber += 1
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			data.HostAddressMap[hostAddress].PinNumberHasHost += t
		} else {
			t := int64(0)
			if pev.Host != "metabitcoin.unknown" {
				t = 1
			}
			data.HostAddressMap[hostAddress] = &MetaSoHostAddress{DataValue: pev.IncrementalValue, Block: blockHeight, Address: pev.Address, MetaId: pev.MetaId, PinNumber: 1, PinNumberHasHost: t, BlockTime: blockTime, Host: pev.Host}
		}
	}
	// data.BlockInfoData.AddressNumber += int64(len(addressMap))
	// data.BlockInfoData.HostNumber += int64(len(hostMap))
	if !isMempool {
		err = pb.SaveBlockPevData(blockHeight, &pevList)
		if err != nil {
			log.Println("Error saving PEV data:", err)
		}
	}
	return 
}