package main

import (
	"embed"
	"fmt"
	"log"
	"manindexer/api"
	"manindexer/basicprotocols/metaname"
	"manindexer/basicprotocols/metaso"
	"manindexer/basicprotocols/mrc721"
	"manindexer/common"
	"manindexer/database/mongodb"
	"manindexer/man"
	"time"
)

// @title           Metaso API
// @version         1.0
// @description     This is a sample API with Swagger documentation.
var (
	//go:embed web/static/* web/template/*
	f embed.FS
)

func main() {
	banner := `
    __  ___  ___     _   __
   /  |/  / /   |   / | / / v0.5.0
  / /|_/ / / /| |  /  |/ / 
 / /  / / / ___ | / /|  /  
/_/  /_/ /_/  |_|/_/ |_/                   
 `
	fmt.Println(banner)
	common.InitConfig("./config.toml")
	cmd := common.Cmd
	fmt.Println("cmd:", cmd)
	man.InitAdapter(common.Chain, common.Db, common.TestNet, common.Server)
	log.Printf("ManIndex,chain=%s,fullnode=%v,test=%s,db=%s,server=%s,config=%s,metaChain=%s", common.Chain, common.Config.Sync.IsFullNode, common.TestNet, common.Db, common.Server, common.ConfigFile, common.Config.Statistics.MetaChainHost)
	if common.Server == "1" {
		go api.Start(f)
	}
	ms := metaso.MetaSo{}
	if common.ModuleExist("metaso") || common.ModuleExist("metaso_pev") {
		metaso.ConnectMongoDb()
	}
	if common.ModuleExist("metaso") {
		ms.SaveSynchBlockedSetting()
		ms.SaveRecommendedAuthor()
		go ms.SynchBlockedSettings()
		go ms.Synchronization()
	}
	go man.ZmqRun()
	if common.ModuleExist("metaso_pev") {
		metaso.PebblePevInit()
		//go ms.SyncPEV()
	}
	if common.ModuleExist("metaname") {
		mn := metaname.MetaName{}
		go mn.Synchronization()
	}
	if common.ModuleExist("mrc721") {
		mrc721 := mrc721.Mrc721{}
		go mrc721.Synchronization()
	}
	go mongodb.FixNullMetaIdPinId()
	for {
		man.IndexerRun(common.TestNet)
		man.CheckNewBlock()
		if common.ModuleExist("metaso_pev") {
			ms.SyncPEV()
		}
		time.Sleep(time.Second * 10)
	}
}
