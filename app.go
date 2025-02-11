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

var (
	//go:embed web/static/* web/template/*
	f embed.FS
)

func main() {
	banner := `
    __  ___  ___     _   __
   /  |/  / /   |   / | / / v0.0.2.5
  / /|_/ / / /| |  /  |/ / 
 / /  / / / ___ | / /|  /  
/_/  /_/ /_/  |_|/_/ |_/                   
 `
	fmt.Println(banner)
	common.InitConfig()
	man.InitAdapter(common.Chain, common.Db, common.TestNet, common.Server)
	log.Printf("ManIndex,chain=%s,test=%s,db=%s,server=%s,config=%s", common.Chain, common.TestNet, common.Db, common.Server, common.ConfigFile)
	if common.Server == "1" {
		go api.Start(f)
	}
	go man.ZmqRun()
	if common.ModuleExist("metaso") {
		ms := metaso.MetaSo{}
		metaso.ConnectMongoDb()
		if common.Config.MetaSo.SyncMode == "db" {
			ms.SyncPin(500)
		}
		go ms.SynchBlockedSettings()
		go ms.Synchronization()
		go ms.SyncPEV()
		go ms.SyncPendingPEVF()
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
