package main

import (
	"context"
	"net/smtp"
	"strings"

	qdb "github.com/rqure/qdb/src"
	"github.com/rqure/qlib/pkg/app"
	"github.com/rqure/qlib/pkg/data"
	"github.com/rqure/qlib/pkg/data/notification"
	"github.com/rqure/qlib/pkg/log"
)

type SmtpConfig struct {
	EmailAddress string
	EmailPwd     string
	Host         string
	Port         string
}

type SmtpWorkerstruct {
	Quit qdb.Signal
}

type SmtpWorker struct {
	store              data.Store
	isLeader           bool
	notificationTokens []data.NotificationToken

	config SmtpConfig

	SmtpWorkerSignals
}

func NewSmtpWorker(store data.Store, config SmtpConfig) *SmtpWorker {
	return &SmtpWorker{
		db:                 db,
		isLeader:           false,
		notificationTokens: []data.NotificationToken{},
		config:             config,
	}
}

func (w *SmtpWorker) OnBecameLeader(context.Context) {
	w.isLeader = true

	w.notificationTokens = append(w.notificationTokens, w.store.Notify(&qdb.DatabaseNotificationConfig{
		Type:  "SmtpController",
		Field: "SendTrigger",
		ContextFields: []string{
			"To",
			"Cc",
			"Subject",
			"Body",
		},
	}, notification.NewCallback(w.ProcessNotification)))
}

func (w *SmtpWorker) OnLostLeadership(context.Context) {
	w.isLeader = false

	for _, token := range w.notificationTokens {
		token.Unbind()
	}

	w.notificationTokens = []data.NotificationToken{}
}

func (w *SmtpWorker) ProcessNotification(ctx context.Context, notification data.Notification) {
	if !w.isLeader {
		return
	}

	log.Info("Received notification: %v", notification)

	from := w.config.EmailAddress
	to := strings.Split(notification.GetContext(0).GetValue().GetString(), ",")
	cc := strings.Split(notification.GetContext(1).GetValue().GetString(), ",")
	subject := notification.GetContext(2).GetValue().GetString()
	body := notification.GetContext(3).GetValue().GetString()
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
			log.Error("Error sending email: %v. Message was: %v", err, message)

			// If we can't send the email, we should quit the application
			// because it may be a networking issue with the container
			w.Quit.Emit()
			return
		}

		log.Info("Email sent successfully")
	}()
}

func (w *SmtpWorker) Init(context.Context, app.Handle) {

}

func (w *SmtpWorker) Deinit(context.Context) {

}

func (w *SmtpWorker) DoWork(context.Context) {

}
