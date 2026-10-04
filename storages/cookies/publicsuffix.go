package cookies

import (
	"strings"

	"golang.org/x/net/publicsuffix"
)

func isPublicSuffix(domain string) bool {
	suffix, _ := publicsuffix.PublicSuffix(domain)
	return strings.EqualFold(suffix, domain)
}

func registrableDomain(host string) string {
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return domain
}
