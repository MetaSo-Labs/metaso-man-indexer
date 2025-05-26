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

	"net/http"
	_ "net/http/pprof"
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
   /  |/  / /   |   / | / / v0.4.18
  / /|_/ / / /| |  /  |/ / 
 / /  / / / ___ | / /|  /  
/_/  /_/ /_/  |_|/_/ |_/                   
 `
	fmt.Println(banner)
	// start pprof
	go func() {
		log.Println("Starting pprof server on :6061")
		if err := http.ListenAndServe(":6061", nil); err != nil {
			log.Fatalf("Pprof server failed to start: %v", err)
		}
	}()
	common.InitConfig("./config.toml")
	man.InitAdapter(common.Chain, common.Db, common.TestNet, common.Server)
	log.Printf("ManIndex,chain=%s,fullnode=%v,test=%s,db=%s,server=%s,config=%s,metaChain=%s", common.Chain, common.Config.Sync.IsFullNode, common.TestNet, common.Db, common.Server, common.ConfigFile, common.Config.Statistics.MetaChainHost)
	if common.Server == "1" {
		go api.Start(f)
	}
	// for {
	// 	time.Sleep(time.Minute * 10)
	// }
	if common.ModuleExist("metaso") && !common.Config.Sync.IsFullNode {
		ms := metaso.MetaSo{}
		metaso.ConnectMongoDb()
		ms.SaveSynchBlockedSetting()
		ms.SaveRecommendedAuthor()
		go ms.SynchBlockedSettings()
		go ms.Synchronization()
	}
	go man.ZmqRun()
	if common.ModuleExist("metaso") {
		ms := metaso.MetaSo{}
		if common.Config.MetaSo.SyncMode == "db" {
			ms.SyncPin(500)
		}
		if common.Config.Sync.IsFullNode {
			go ms.SyncPEV()
		}
		//go ms.SyncPendingPEVF()
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
		time.Sleep(time.Second * 10)
	}
}
