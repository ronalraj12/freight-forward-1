package main

import (
	"github.com/RemoteState/yourdaily-server/cronJobs"
	"github.com/RemoteState/yourdaily-server/database"
	"github.com/RemoteState/yourdaily-server/server"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

func InitiateCronJobs() error {
	logrus.Infof("intiating cronJobs jobs")
	checkAndUpdateOrderStatus := cron.New()
	err := checkAndUpdateOrderStatus.AddFunc("@every 10s", func() {
		cronJobs.CronFuncToCheckOrderStatus()
	})
	if err != nil {
		logrus.Errorf("cronJobs job intiation failed %v", err)
		return err
	}
	checkAndUpdateOrderStatus.Start()

	moveScheduledOrdersToNow := cron.NewWithLocation(time.Local)
	err = moveScheduledOrdersToNow.AddFunc("@hourly", func() {
		logrus.Infof("moving orders")
		cronJobs.MoveScheduledOrders()
	})
	if err != nil {
		logrus.Errorf("cronJobs job(move scheduled orders to now) intiation failed %v", err)
		return err
	}
	moveScheduledOrdersToNow.Start()

	initiateScheduledOrder := cron.NewWithLocation(time.Local)
	err = moveScheduledOrdersToNow.AddFunc("@every 5s", func() {
		logrus.Infof("intiating orders")
		cronJobs.InitiateScheduledOrder()
	})
	if err != nil {
		logrus.Errorf("cronJobs job initiateScheduledOrder intiation failed %v", err)
		return err
	}
	initiateScheduledOrder.Start()

	logrus.Infof("cronJobs job initiation successfull ")
	return nil
}

func main() {
	if err := database.ConnectAndMigrate(os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_USER_NAME"),
		os.Getenv("DB_PASSWORD"),
		database.SSLModeDisable); err != nil {
		logrus.Panicf("Failed to initialize and migrate database with error: %+v", err)
	}

	logrus.Print("migration successful!!")
	err := InitiateCronJobs()
	if err != nil {
		logrus.Error("error form cronJobs job", err)
	}

	// create server instance
	srv := server.SetupRoutes()

	logrus.Print("Server started at ", os.Getenv("SERVER_HOST_PORT"))
	if err := srv.Run(":" + os.Getenv("SERVER_HOST_PORT")); err != nil {
		logrus.Panicf("Failed to run server with error: %+v", err)
	}
}
