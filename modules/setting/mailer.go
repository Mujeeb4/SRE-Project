// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net"
	"net/mail"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"

	shellquote "github.com/kballard/go-shellquote"
)

// Mailer represents mail service.
type Mailer struct {
	// Mailer
	Name                 string `ini:"NAME"`
	From                 string `ini:"FROM"`
	EnvelopeFrom         string `ini:"ENVELOPE_FROM"`
	OverrideEnvelopeFrom bool   `ini:"-"`
	FromName             string `ini:"-"`
	FromEmail            string `ini:"-"`
	SendAsPlainText      bool   `ini:"SEND_AS_PLAIN_TEXT"`
	SubjectPrefix        string `ini:"SUBJECT_PREFIX"`

	// SMTP sender
	Protocol             string `ini:"PROTOCOL"`
	SMTPAddr             string `ini:"SMTP_ADDR"`
	SMTPPort             string `ini:"SMTP_PORT"`
	User                 string `ini:"USER"`
	Passwd               string `ini:"PASSWD"`
	EnableHelo           bool   `ini:"ENABLE_HELO"`
	HeloHostname         string `ini:"HELO_HOSTNAME"`
	ForceTrustServerCert bool   `ini:"FORCE_TRUST_SERVER_CERT"`
	UseClientCert        bool   `ini:"USE_CLIENT_CERT"`
	ClientCertFile       string `ini:"CLIENT_CERT_FILE"`
	ClientKeyFile        string `ini:"CLIENT_KEY_FILE"`

	// Sendmail sender
	SendmailPath        string        `ini:"SENDMAIL_PATH"`
	SendmailArgs        []string      `ini:"-"`
	SendmailTimeout     time.Duration `ini:"SENDMAIL_TIMEOUT"`
	SendmailConvertCRLF bool          `ini:"SENDMAIL_CONVERT_CRLF"`
}

// MailService the global mailer
var MailService *Mailer

func loadMailsFrom(rootCfg ConfigProvider) {
	loadMailerFrom(rootCfg)
	loadRegisterMailFrom(rootCfg)
	loadNotifyMailFrom(rootCfg)
	loadIncomingEmailFrom(rootCfg)
}

func loadMailerFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("mailer")
	// Check mailer setting.
	if !sec.Key("ENABLED").MustBool() {
		return
	}

	// TODO: Remove before 1.19.0
	if sec.HasKey("PROTOCOL") && sec.Key("PROTOCOL").String() == "smtp+startls" {
		log.Error("Deprecated fallback `[mailer]` `PROTOCOL = smtp+startls` present. Use `[mailer]` `PROTOCOL = smtp+starttls`` instead. This fallback will be removed in v1.19.0")
		sec.Key("PROTOCOL").SetValue("smtp+starttls")
	}

	// Set default values & validate
	sec.Key("NAME").MustString(AppName)
	sec.Key("PROTOCOL").In("", []string{"smtp", "smtps", "smtp+starttls", "smtp+unix", "sendmail", "dummy"})
	sec.Key("SMTP_ADDR").MustString("")
	sec.Key("SMTP_PORT").MustString("")
	sec.Key("ENABLE_HELO").MustBool(true)
	sec.Key("FORCE_TRUST_SERVER_CERT").MustBool(false)
	sec.Key("USE_CLIENT_CERT").MustBool(false)
	sec.Key("CLIENT_CERT_FILE").MustString("custom/mailer/cert.pem")
	sec.Key("CLIENT_KEY_FILE").MustString("custom/mailer/key.pem")
	sec.Key("SENDMAIL_PATH").MustString("sendmail")
	sec.Key("SENDMAIL_TIMEOUT").MustDuration(5 * time.Minute)
	sec.Key("SENDMAIL_CONVERT_CRLF").MustBool(true)
	sec.Key("SEND_AS_PLAIN_TEXT").MustBool(false)
	sec.Key("FROM").MustString(sec.Key("USER").String())

	// Now map the values on to the MailService
	MailService = &Mailer{}
	if err := sec.MapTo(MailService); err != nil {
		log.Fatal("Unable to map [mailer] section on to MailService. Error: %v", err)
	}

	// Infer SMTPPort if not set
	if MailService.SMTPPort == "" {
		switch MailService.Protocol {
		case "smtp":
			MailService.SMTPPort = "25"
		case "smtps":
			MailService.SMTPPort = "465"
		case "smtp+starttls":
			MailService.SMTPPort = "587"
		}
	}

	// Infer Protocol
	if MailService.Protocol == "" {
		if strings.ContainsAny(MailService.SMTPAddr, "/\\") {
			MailService.Protocol = "smtp+unix"
		} else {
			switch MailService.SMTPPort {
			case "25":
				MailService.Protocol = "smtp"
			case "465":
				MailService.Protocol = "smtps"
			case "587":
				MailService.Protocol = "smtp+starttls"
			default:
				log.Error("unable to infer unspecified mailer.PROTOCOL from mailer.SMTP_PORT = %q, assume using smtps", MailService.SMTPPort)
				MailService.Protocol = "smtps"
				if MailService.SMTPPort == "" {
					MailService.SMTPPort = "465"
				}
			}
		}
	}

	// we want to warn if users use SMTP on a non-local IP;
	// we might as well take the opportunity to check that it has an IP at all
	// This check is not needed for sendmail
	switch MailService.Protocol {
	case "sendmail":
		var err error
		MailService.SendmailArgs, err = shellquote.Split(sec.Key("SENDMAIL_ARGS").String())
		if err != nil {
			log.Error("Failed to parse Sendmail args: '%s' with error %v", sec.Key("SENDMAIL_ARGS").String(), err)
		}
	case "smtp", "smtps", "smtp+starttls", "smtp+unix":
		ips := tryResolveAddr(MailService.SMTPAddr)
		if MailService.Protocol == "smtp" {
			for _, ip := range ips {
				if !ip.IsLoopback() {
					log.Warn("connecting over insecure SMTP protocol to non-local address is not recommended")
					break
				}
			}
		}
	case "dummy": // just mention and do nothing
	}

	if MailService.From != "" {
		parsed, err := mail.ParseAddress(MailService.From)
		if err != nil {
			log.Fatal("Invalid mailer.FROM (%s): %v", MailService.From, err)
		}
		MailService.FromName = parsed.Name
		MailService.FromEmail = parsed.Address
	} else {
		log.Error("no mailer.FROM provided, email system may not work.")
	}

	switch MailService.EnvelopeFrom {
	case "":
		MailService.OverrideEnvelopeFrom = false
	case "<>":
		MailService.EnvelopeFrom = ""
		MailService.OverrideEnvelopeFrom = true
	default:
		parsed, err := mail.ParseAddress(MailService.EnvelopeFrom)
		if err != nil {
			log.Fatal("Invalid mailer.ENVELOPE_FROM (%s): %v", MailService.EnvelopeFrom, err)
		}
		MailService.OverrideEnvelopeFrom = true
		MailService.EnvelopeFrom = parsed.Address
	}

	log.Info("Mail Service Enabled")
}

func loadRegisterMailFrom(rootCfg ConfigProvider) {
	if !rootCfg.Section("service").Key("REGISTER_EMAIL_CONFIRM").MustBool() {
		return
	} else if MailService == nil {
		log.Warn("Register Mail Service: Mail Service is not enabled")
		return
	}
	Service.RegisterEmailConfirm = true
	log.Info("Register Mail Service Enabled")
}

func loadNotifyMailFrom(rootCfg ConfigProvider) {
	if !rootCfg.Section("service").Key("ENABLE_NOTIFY_MAIL").MustBool() {
		return
	} else if MailService == nil {
		log.Warn("Notify Mail Service: Mail Service is not enabled")
		return
	}
	Service.EnableNotifyMail = true
	log.Info("Notify Mail Service Enabled")
}

func tryResolveAddr(addr string) []net.IP {
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		addr = addr[1 : len(addr)-1]
	}
	ip := net.ParseIP(addr)
	if ip != nil {
		ips := make([]net.IP, 1)
		ips[0] = ip
		return ips
	}
	ips, err := net.LookupIP(addr)
	if err != nil {
		log.Warn("could not look up mailer.SMTP_ADDR: %v", err)
		return make([]net.IP, 0)
	}
	return ips
}
