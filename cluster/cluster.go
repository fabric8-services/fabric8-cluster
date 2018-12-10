package cluster

import (
	"net/url"

	"github.com/fabric8-services/fabric8-common/httpsupport"
)

const (
	// OSD the OpenShift Dedicated type of cluster
	OSD = "OSD"
	// OCP the OpenShift On-Premise type of Cluster
	OCP = "OCP"
	// OSO the OpenShoft online type of cluster
	OSO = "OSO"
)

// ConvertAPIURL converts the given `apiURL` by adding the new prefix (or subdomain) and a path, with a trailing slash
// eg: ConvertAPIURL("https://foo.com", "api", "some/path") gives "https://api.foo.com/some/path/"
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
	return httpsupport.AddTrailingSlashToURL(newURL.String()), nil
}
