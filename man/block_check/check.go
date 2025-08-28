/*
对一些重要的数据做重新检查，比如用户信息
*/
package blockcheck

import (
	"fmt"
	"log"
	"manindexer/common"
	"manindexer/man"
	"manindexer/pin"
	"strconv"
	"strings"
	"time"
)

type CheckChain struct {
	ChainName string
	From      int64
	To        int64
}

func CheckRun() {
	for {
		// 进行区块检查
		StartCheck()
		time.Sleep(time.Second * 20)
	}
}
func StartCheck() {
	var checkChains []CheckChain
	chains := strings.Split(common.Chain, ",")
	for _, chain := range chains {
		check := CheckChain{ChainName: chain}
		key := "check_height_" + chain
		v, err := common.LoadFromDictDB(key)
		if err == nil && v != nil {
			h, err := strconv.ParseInt(string(v), 10, 64)
			if err == nil {
				check.From = h + 1
			}
		}
		check.To = man.ChainAdapter[chain].GetBestHeight()
		checkChains = append(checkChains, check)
	}
	if len(checkChains) <= 0 {
		return
	}
	DoCheck(checkChains)
}
func DoCheck(checkChains []CheckChain) {
	for _, check := range checkChains {
		if check.From <= 0 || check.To <= 0 || check.From > check.To {
			continue
		}
		log.Println(check.ChainName, "Checking blocks:", check.From, "to", check.To)
		for height := check.From; height <= check.To; height++ {
			pins, _, _ := man.IndexerAdapter[check.ChainName].CatchPins(height)
			//fmt.Println(height, ">>>num:", len(pins))
			// 进行区块数据的检查
			if len(pins) == 0 {
				common.SaveToDictDB("check_height_"+check.ChainName, []byte(strconv.FormatInt(height, 10)))
				continue
			}

			userMap := make(map[string]*pin.MetaIdInfo)
			for _, pinNode := range pins {
				CheckUserInfoPath(pinNode, &userMap)
			}
			SaveUserInfo(userMap)
			common.SaveToDictDB("check_height_"+check.ChainName, []byte(strconv.FormatInt(height, 10)))
		}
	}
}
func SaveUserInfo(userMap map[string]*pin.MetaIdInfo) {
	if len(userMap) <= 0 {
		return
	}
	// for addr, info := range userMap {
	// 	//fmt.Println(addr, info)

	// }
	man.DbAdapter.BatchUpsertMetaIdInfo(userMap)

}
func CheckUserInfoPath(pinNode *pin.PinInscription, userMap *map[string]*pin.MetaIdInfo) {
	cache := false
	// 进行区块数据的检查
	if pinNode.Operation == "modify" {
		//fmt.Println("tx:", pinNode.Id, string(pinNode.ContentBody))
		pinId := pinNode.Path
		modifyPath := man.GetModifyPath(pinId)
		if modifyPath != "" {
			pinNode.Path = modifyPath
		}
	}
	switch pinNode.Path {
	case "/info/name":
		if v, ok := (*userMap)[pinNode.Address]; ok {
			v.Name = string(pinNode.ContentBody)
			v.NameId = pinNode.Id
			v.Address = pinNode.Address
		} else {
			(*userMap)[pinNode.Address] = &pin.MetaIdInfo{
				Name:    string(pinNode.ContentBody),
				NameId:  pinNode.Id,
				Address: pinNode.Address,
			}
		}
		cache = true
	case "/info/avatar":
		if v, ok := (*userMap)[pinNode.Address]; ok {
			v.Avatar = string(pinNode.ContentBody)
			v.AvatarId = pinNode.Id
			v.Address = pinNode.Address
		} else {
			(*userMap)[pinNode.Address] = &pin.MetaIdInfo{
				Avatar:   string(pinNode.ContentBody),
				AvatarId: pinNode.Id,
				Address:  pinNode.Address,
			}
		}
		cache = true
	case "/info/bio":
		if v, ok := (*userMap)[pinNode.Address]; ok {
			v.Bio = string(pinNode.ContentBody)
			v.BioId = pinNode.Id
			v.Address = pinNode.Address
		} else {
			(*userMap)[pinNode.Address] = &pin.MetaIdInfo{
				Bio:     string(pinNode.ContentBody),
				BioId:   pinNode.Id,
				Address: pinNode.Address,
			}
		}
		cache = true
	case "/info/background":
		if v, ok := (*userMap)[pinNode.Address]; ok {
			v.Background = string(pinNode.ContentBody)
			v.Address = pinNode.Address
		} else {
			(*userMap)[pinNode.Address] = &pin.MetaIdInfo{
				Background: string(pinNode.ContentBody),
				Address:    pinNode.Address,
			}
		}
	case "/info/chatpubkey":
		if v, ok := (*userMap)[pinNode.Address]; ok {
			v.ChatPubKey = string(pinNode.ContentBody)
			v.Address = pinNode.Address
		} else {
			(*userMap)[pinNode.Address] = &pin.MetaIdInfo{
				ChatPubKey: string(pinNode.ContentBody),
				Address:    pinNode.Address,
			}
		}
		cache = true
	}
	if cache {
		fmt.Println("Caching user info for:", pinNode.Address, pinNode.Path, "Content:", string(pinNode.ContentBody))
		man.SetCache(pinNode)
	}

}
