package cluster

import (
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"net/url"
)

const (
	OSD = "OSD"
	OCP = "OCP"
	OSO = "OSO"
)

func ConvertAPIURL(apiURL, newPrefix, newPath string) (string, error) {
	newURL, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	newHost, err := httpsupport.ReplaceDomainPrefix(newURL.Host, newPrefix)
	if err != nil {
		return "", err
	}
	newURL.Host = newHost
	newURL.Path = newPath
	return newURL.String(), nil
}
