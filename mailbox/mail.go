package mailbox

import (
	"fmt"
	"net/smtp"
	"strings"

	l4g "github.com/alecthomas/log4go"
)

type Config struct {
	User          string `xml:"user"`
	Password      string `xml:"password"`
	Server        string `xml:"server"`
	To            string `xml:"to"`
	SubjectPrefix string `xml:"subject_prefix"`
}

type MailBox struct {
	cfg *Config
}

func NewMailBox(cfg *Config) (*MailBox, error) {
	mb := &MailBox{
		cfg: cfg,
	}

	if err := mb.test(); nil != err {
		return nil, err
	}
	return mb, nil
}

func (mb *MailBox) test() error {
	c, err := smtp.Dial(mb.cfg.Server)
	if err != nil {
		return err
	}
	defer c.Close()
	//hp := strings.Split(mb.cfg.Server, ":")
	//auth := smtp.PlainAuth("", mb.cfg.User, mb.cfg.Password, hp[0])
	//if err = c.Auth(auth); err != nil {
	//	return err
	//}

	return nil
}

func (mb *MailBox) SendMail(subject string, body string) error {
	err := SendMail(mb.cfg.User,
		mb.cfg.Password,
		mb.cfg.Server,
		mb.cfg.To,
		fmt.Sprintf("%s%s", mb.cfg.SubjectPrefix, subject),
		body,
		"")
	return err
}

func SendMail(user, password, host, to, subject, body, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + user + "\r\nSubject: " + subject + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	l4g.Debug("[mailbox] SendMail [%s] [%s]", subject, body)
	return err
}
