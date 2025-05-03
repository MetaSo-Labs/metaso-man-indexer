package main

import (
	"context"
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
   /  |/  / /   |   / | / / v0.5.8
  / /|_/ / / /| |  /  |/ / 
 / /  / / / ___ | / /|  /  
/_/  /_/ /_/  |_|/_/ |_/                   
 `
	fmt.Println(banner)
	common.InitConfig()
	man.InitAdapter(common.Chain, common.Db, common.TestNet, common.Server)
	log.Printf("ManIndex,chain=%s,test=%s,db=%s,server=%s,config=%s,metaChain=%s", common.Chain, common.TestNet, common.Db, common.Server, common.ConfigFile, common.Config.Statistics.MetaChainHost)
	if common.Server == "1" {
		go api.Start(f)
	}
	go man.ZmqRun()
	man.IndexerRun(common.TestNet)
	man.CheckNewBlock()
	if common.ModuleExist("metaso") {
		ms := metaso.MetaSo{}
		metaso.ConnectMongoDb()
		if common.Config.MetaSo.SyncMode == "db" {
			ms.SyncPin(500)
		}
		go ms.SynchBlockedSettings()
		go ms.Synchronization()
		go ms.SyncPEV()
		//go ms.SyncPendingPEVF()
	}
	if common.ModuleExist("metaname") {
		mn := metaname.MetaName{}
		go mn.Synchronization()
	}
	if common.ModuleExist("mrc721") {
		mrc721c := mrc721.Mrc721{}
		mrc721.ConnectMongoDb()
		go mrc721c.Synchronization()
		go mrc721c.SynchronizationAddress()
	}
	go mongodb.FixNullMetaIdPinId()
	// for {
	// 	man.IndexerRun(common.TestNet)
	// 	man.CheckNewBlock()
	// 	time.Sleep(time.Second * 10)
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			man.IndexerRun(common.TestNet)
			man.CheckNewBlock()
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
	}
}
