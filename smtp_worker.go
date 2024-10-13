package main

import (
	"net/smtp"
	"strings"

	qdb "github.com/rqure/qdb/src"
)

type SmtpConfig struct {
	EmailAddress string
	EmailPwd     string
	Host         string
	Port         string
}

type SmtpWorkerSignals struct {
	Quit qdb.Signal
}

type SmtpWorker struct {
	db                 qdb.IDatabase
	isLeader           bool
	notificationTokens []qdb.INotificationToken

	config SmtpConfig

	Signals SmtpWorkerSignals
}

func NewSmtpWorker(db qdb.IDatabase, config SmtpConfig) *SmtpWorker {
	return &SmtpWorker{
		db:                 db,
		isLeader:           false,
		notificationTokens: []qdb.INotificationToken{},
		config:             config,
	}
}

func (w *SmtpWorker) OnBecameLeader() {
	w.isLeader = true

	w.notificationTokens = append(w.notificationTokens, w.db.Notify(&qdb.DatabaseNotificationConfig{
		Type:  "SmtpController",
		Field: "SendTrigger",
		ContextFields: []string{
			"To",
			"Cc",
			"Subject",
			"Body",
		},
	}, qdb.NewNotificationCallback(w.ProcessNotification)))
}

func (w *SmtpWorker) OnLostLeadership() {
	w.isLeader = false

	for _, token := range w.notificationTokens {
		token.Unbind()
	}

	w.notificationTokens = []qdb.INotificationToken{}
}

func (w *SmtpWorker) ProcessNotification(notification *qdb.DatabaseNotification) {
	if !w.isLeader {
		return
	}

	qdb.Info("[SmtpWorker::ProcessNotification] Received notification: %v", notification)

	from := w.config.EmailAddress
	to := strings.Split(qdb.ValueCast[*qdb.String](notification.Context[0].Value).Raw, ",")
	cc := strings.Split(qdb.ValueCast[*qdb.String](notification.Context[1].Value).Raw, ",")
	subject := qdb.ValueCast[*qdb.String](notification.Context[2].Value).Raw
	body := qdb.ValueCast[*qdb.String](notification.Context[3].Value).Raw
	allRecipients := append(to, cc...)
	message := []byte(
		"From: " + from + "\n" +
			"To: " + strings.Join(to, ",") + "\n" +
			"Cc: " + strings.Join(cc, ",") + "\n" +
			"Subject: " + subject + "\n\n" +
			body,
	)

	go func() {
		auth := smtp.PlainAuth("", w.config.EmailAddress, w.config.EmailPwd, w.config.Host)

		err := smtp.SendMail(
			w.config.Host+":"+w.config.Port,
			auth,
			from,
			allRecipients,
			message,
		)

		if err != nil {
			qdb.Error("[SmtpWorker::ProcessNotification] Error sending email: %v. Message was: %v", err, message)
			w.Signals.Quit.Emit()
			return
		}

		qdb.Info("[SmtpWorker::ProcessNotification] Email sent successfully")
	}()
}

func (w *SmtpWorker) Init() {

}

func (w *SmtpWorker) Deinit() {

}

func (w *SmtpWorker) DoWork() {

}
