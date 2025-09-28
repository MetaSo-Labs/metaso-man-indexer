package main

import (
	"encoding/json"
	"fmt"
	"manindexer/blockfile"
	"manindexer/pin"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	blockfile.InitBlockFileDb()
	defer blockfile.CloseBlockFileDb()
	blockfile.InitBlockFile()
	// err := blockfile.DownloadFile("mvc", 139410)
	// if err != nil {
	// 	t.Errorf("DownloadFile failed: %v", err)
	// }
	err := blockfile.DoSync(3)
	if err != nil {
		t.Errorf("DoSync failed: %v", err)
	}
}
func TestLoadBlockFile(t *testing.T) {
	blockfile.InitBlockFileDb()
	defer blockfile.CloseBlockFileDb()
	blocks, err := blockfile.LoadFBlockPart("mvc", 139410, 0)
	if err != nil {
		t.Errorf("LoadBlockFile failed: %v", err)
	}
	fmt.Println("-------")
	if len(blocks) == 0 {
		t.Errorf("No blocks loaded")
	}
	fmt.Printf("Loaded %d blocks\n", len(blocks))
	for i, block := range blocks[0:3] {
		var pinNode pin.PinInscription
		err := json.Unmarshal(block, &pinNode)
		if err != nil {
			t.Errorf("Unmarshal block %d failed: %v", i, err)
		}
		fmt.Println("Body:", string(pinNode.ContentBody), pinNode.Host, pinNode.Timestamp)
	}
}

func TestQueryKeysByTimeRange(t *testing.T) {
	blockfile.InitBlockFileDb()
	defer blockfile.CloseBlockFileDb()
	startTs := int64(1758006046)
	endTs := int64(1758006047)

	keys, err := blockfile.QueryKeysByTimeRange(startTs, endTs)
	if err != nil {
		t.Errorf("QueryKeysByTimeRange failed: %v", err)
	}
	fmt.Printf("Found %d keys between %d and %d:\n", len(keys), startTs, endTs)
	for _, key := range keys {
		fmt.Println(key)
	}
}
func TestQueryPinsByTimeRange(t *testing.T) {
	blockfile.InitBlockFileDb()
	defer blockfile.CloseBlockFileDb()
	startTs := int64(1758006046)
	endTs := int64(1758006047)
	host := []string{}
	protocol := []string{}
	//host := []string{"bc1p20k3x2c4mglfxr5wa5sgtgechwstpld80kru2cg4gmm4urvuaqqsvapxu0"}
	//protocol := []string{"/protocols/simplefilegroupchat"}

	pins, err := blockfile.QueryPinsByTimeRange(startTs, endTs, host, protocol)
	if err != nil {
		t.Errorf("QueryPinsByTimeRange failed: %v", err)
	}
	fmt.Printf("Found %d pins between %d and %d:\n", len(pins), startTs, endTs)
	var hostMap = make(map[string]int)
	var pathMap = make(map[string]int)
	for _, pin := range pins {
		hostMap[pin.Host]++
		pathMap[pin.Path]++
		//fmt.Printf("Host: %s, Chain: %s, Height: %d, Timestamp: %d\n", pin.Host, pin.ChainName, pin.GenesisHeight, pin.Timestamp)
	}
	for h := range hostMap {
		fmt.Println("Host:", h, hostMap[h])
	}
	for p := range pathMap {
		fmt.Println("Path:", p, pathMap[p])
	}
}
