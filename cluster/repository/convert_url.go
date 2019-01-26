package repository

import (
	"net/url"

	"github.com/fabric8-services/fabric8-common/errors"

	"github.com/fabric8-services/fabric8-common/httpsupport"
)

// ConvertAPIURL converts the given `apiURL` by adding the new prefix (or subdomain) and a path
// eg: ConvertAPIURL("https://foo.com", "api", "some/path") gives "https://api.foo.com/some/path"
func ConvertAPIURL(apiURL, newPrefix, newPath string) (string, error) {
	newURL, err := url.Parse(apiURL)
	if err != nil {
		return "", errors.NewBadParameterErrorFromString(err.Error())
	}
	newHost, err := httpsupport.ReplaceDomainPrefix(newURL.Host, newPrefix)
	if err != nil {
		return "", err
	}
	newURL.Host = newHost
	newURL.Path = newPath
	return newURL.String(), nil
}
