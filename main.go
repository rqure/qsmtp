package main

import (
	"os"

	qdb "github.com/rqure/qdb/src"
)

func getDatabaseAddress() string {
	addr := os.Getenv("QDB_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}

	return addr
}

func getEmailAddress() string {
	return os.Getenv("QDB_EMAIL_ADDRESS")
}

func getEmailPassword() string {
	return os.Getenv("QDB_EMAIL_PASSWORD")
}

func getEmailHost() string {
	host := os.Getenv("QDB_EMAIL_HOST")
	if host == "" {
		host = "smtp.gmail.com"
	}

	return host
}

func getEmailPort() string {
	port := os.Getenv("QDB_EMAIL_PORT")
	if port == "" {
		port = "587"
	}

	return port
}

func main() {
	db := qdb.NewRedisDatabase(qdb.RedisDatabaseConfig{
		Address: getDatabaseAddress(),
	})

	dbWorker := qdb.NewDatabaseWorker(db)
	leaderElectionWorker := qdb.NewLeaderElectionWorker(db)
	smtpWorker := NewSmtpWorker(db, SmtpConfig{
		EmailAddress: getEmailAddress(),
		EmailPwd:     getEmailPassword(),
		Host:         getEmailHost(),
		Port:         getEmailPort(),
	})
	schemaValidator := qdb.NewSchemaValidator(db)

	schemaValidator.AddEntity("Root", "SchemaUpdateTrigger")
	schemaValidator.AddEntity("SmtpController", "To", "Cc", "Subject", "Body", "SendTrigger")

	dbWorker.Signals.SchemaUpdated.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	dbWorker.Signals.Connected.Connect(qdb.Slot(schemaValidator.ValidationRequired))
	leaderElectionWorker.AddAvailabilityCriteria(func() bool {
		return schemaValidator.IsValid()
	})

	dbWorker.Signals.Connected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseConnected))
	dbWorker.Signals.Disconnected.Connect(qdb.Slot(leaderElectionWorker.OnDatabaseDisconnected))

	leaderElectionWorker.Signals.BecameLeader.Connect(qdb.Slot(smtpWorker.OnBecameLeader))
	leaderElectionWorker.Signals.LosingLeadership.Connect(qdb.Slot(smtpWorker.OnLostLeadership))

	// Create a new application configuration
	config := qdb.ApplicationConfig{
		Name: "smtp",
		Workers: []qdb.IWorker{
			dbWorker,
			leaderElectionWorker,
			smtpWorker,
		},
	}

	// Create a new application
	app := qdb.NewApplication(config)

	// Execute the application
	app.Execute()
}
