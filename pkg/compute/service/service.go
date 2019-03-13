package service

import (
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	_ "yunion.io/x/onecloud/pkg/compute/regiondrivers"
	"yunion.io/x/onecloud/pkg/compute/skus"
	_ "yunion.io/x/onecloud/pkg/compute/storagedrivers"
	_ "yunion.io/x/onecloud/pkg/compute/tasks"
	_ "yunion.io/x/onecloud/pkg/util/aliyun/provider"
	_ "yunion.io/x/onecloud/pkg/util/aws/provider"
	_ "yunion.io/x/onecloud/pkg/util/azure/provider"
	_ "yunion.io/x/onecloud/pkg/util/esxi/provider"
	_ "yunion.io/x/onecloud/pkg/util/huawei/provider"
	_ "yunion.io/x/onecloud/pkg/util/openstack/provider"
	_ "yunion.io/x/onecloud/pkg/util/qcloud/provider"
)

func StartService() {

	opts := &options.Options
	commonOpts := &options.Options.CommonOptions
	dbOpts := &options.Options.DBOptions
	cloudcommon.ParseOptions(opts, os.Args, "region.conf", "compute")

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		commonOpts.Port = opts.PortV2
	}

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(commonOpts, true)

	InitHandlers(app)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	err := models.InitDB()
	if err != nil {
		log.Errorf("InitDB fail: %s", err)
	}

	err = setInfluxdbRetentionPolicy()
	if err != nil {
		log.Errorf("setInfluxdbRetentionPolicy fail: %s", err)
	}

	cron := cronman.GetCronJobManager(true)
	cron.AddJob1("CleanPendingDeleteServers", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.GuestManager.CleanPendingDeleteServers)
	cron.AddJob1("CleanPendingDeleteDisks", time.Duration(opts.PendingDeleteCheckSeconds)*time.Second, models.DiskManager.CleanPendingDeleteDisks)
	cron.AddJob1("CleanPendingDeleteLoadbalancers", time.Duration(opts.LoadbalancerPendingDeleteCheckInterval)*time.Second, models.LoadbalancerAgentManager.CleanPendingDeleteLoadbalancers)
	cron.AddJob1("CleanExpiredPrepaidServers", time.Duration(opts.PrepaidExpireCheckSeconds)*time.Second, models.GuestManager.DeleteExpiredPrepaidServers)
	cron.AddJob1("StartHostPingDetectionTask", time.Duration(opts.HostOfflineDetectionInterval)*time.Second, models.HostManager.PingDetectionTask)

	cron.AddJob2("AutoDiskSnapshot", opts.AutoSnapshotDay, opts.AutoSnapshotHour, 0, 0, models.DiskManager.AutoDiskSnapshot, false)
	cron.AddJob2("SyncSkus", opts.SyncSkusDay, opts.SyncSkusHour, 0, 0, skus.SyncSkus, true)

	cron.Start()
	defer cron.Stop()

	cloudcommon.ServeForever(app, commonOpts)
}
