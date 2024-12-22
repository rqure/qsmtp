package main

import (
	"os"

	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/app/workers"
	"github.com/rqure/qlib/pkg/data/store"
)

func getDatabaseAddress() string {
	addr := os.Getenv("Q_ADDR")
	if addr == "" {
		addr = "ws://webgateway:20000/ws"
	}

	return addr
}

func getEmailAddress() string {
	return os.Getenv("Q_EMAIL_ADDRESS")
}

func getEmailPassword() string {
	return os.Getenv("Q_EMAIL_PASSWORD")
}

func getEmailHost() string {
	host := os.Getenv("Q_EMAIL_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}

	return host
}

func getEmailPort() string {
	port := os.Getenv("Q_EMAIL_PORT")
	if port == "" {
		port = "587"
	}

	return port
}

func main() {
	db := store.NewWeb(store.WebConfig{
		Address: getDatabaseAddress(),
	})

	storeWorker := workers.NewStore(db)
	leadershipWorker := workers.NewLeadership(db)
	smtpWorker := NewSmtpWorker(db, SmtpConfig{
		EmailAddress: getEmailAddress(),
		EmailPwd:     getEmailPassword(),
		Host:         getEmailHost(),
		Port:         getEmailPort(),
	})
	schemaValidator := leadershipWorker.GetEntityFieldValidator()

	schemaValidator.RegisterEntityFields("Root", "SchemaUpdateTrigger")
	schemaValidator.RegisterEntityFields("SmtpController", "To", "Cc", "Subject", "Body", "SendTrigger")

	storeWorker.Connected.Connect(leadershipWorker.OnStoreConnected)
	storeWorker.Disconnected.Connect(leadershipWorker.OnStoreDisconnected)

	leadershipWorker.BecameLeader().Connect(smtpWorker.OnBecameLeader)
	leadershipWorker.LosingLeadership().Connect(smtpWorker.OnLostLeadership)

	a := app.NewApplication("smtp")
	a.AddWorker(storeWorker)
	a.AddWorker(leadershipWorker)
	a.AddWorker(smtpWorker)
	a.Execute()
}
