package model

import "html/template"

type EmailData struct {
	To        string
	Body      string
	SMTPHost  string
	SMTPPort  int
	FromName  string
	FromEmail string
	Username  string
	Password  string
	Subject   string
}

type EmailBodyData struct {
	Code            string
	ResetURL        template.URL
	ResetRequestURL template.URL
	Year            int
	Expired         int
}
