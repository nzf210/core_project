package main

import "strings"

func formatPhoneToWAJID(phone string) string {
	p := strings.TrimSpace(phone)
	if strings.HasPrefix(p, "0") {
		p = "62" + p[1:]
	} else if strings.HasPrefix(p, "+") {
		p = p[1:]
	}
	return p + "@s.whatsapp.net"
}
