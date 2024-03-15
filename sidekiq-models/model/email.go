package model

type Email struct {
	Sender   string
	Receiver string
	Header   string
	Subject  string
	HtmlBody string
	TextBody string
}
