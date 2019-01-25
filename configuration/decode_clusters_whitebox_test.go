package configuration

import (
	"testing"

	uuid "github.com/satori/go.uuid"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-common/resource"
)

func TestDecodeClusters(t *testing.T) {

	resource.Require(t, resource.UnitTest)

	t.Run("ok", func(t *testing.T) {
		// given
		data := []map[string]interface{}{
			{
				"name":                            "cluster1",
				"api-url":                         "http://cluster1",
				"console-url":                     "http://console.cluster1",
				"logging-url":                     "http://logging.cluster1",
				"metrics-url":                     "http://metrics.cluster1",
				"app-dns":                         "appdns1",
				"auth-client-id":                  "authclientID1",
				"auth-client-secret":              "authclientsecret1",
				"auth-client-default-scope":       "defaultscope1",
				"capacity-exhausted":              true,
				"service-account-username":        "udername1",
				"service-account-token":           "token1",
				"service-account-token-encrypted": true,
				"token-provider-id":               "provider1",
				"type":                            "OSD",
			},
			{
				"name":                      "cluster2",
				"api-url":                   "http://cluster2",
				"app-dns":                   "appdns2",
				"auth-client-id":            "authclientID2",
				"auth-client-secret":        "authclientsecret2",
				"auth-client-default-scope": "defaultscope2",
				"service-account-username":  "udername2",
				"service-account-token":     "token2",
				"token-provider-id":         "provider2",
			},
		}

		// when
		result, err := decodeClusters(data)
		// then
		require.NoError(t, err)
		require.Len(t, result, 2)
		// a cluster with all optional fields explicitely set
		assert.Equal(t, repository.Cluster{
			ClusterID:         uuid.Nil,
			Name:              "cluster1",
			URL:               "http://cluster1",
			ConsoleURL:        "http://console.cluster1",
			LoggingURL:        "http://logging.cluster1",
			MetricsURL:        "http://metrics.cluster1",
			AppDNS:            "appdns1",
			AuthClientID:      "authclientID1",
			AuthClientSecret:  "authclientsecret1",
			AuthDefaultScope:  "defaultscope1",
			CapacityExhausted: true,
			SAUsername:        "udername1",
			SAToken:           "token1",
			SATokenEncrypted:  true,
			TokenProviderID:   "provider1",
			Type:              "OSD",
		}, result[0])
		// a cluster with all fields optional fields unspecified
		assert.Equal(t, repository.Cluster{
			ClusterID:         uuid.Nil,
			Name:              "cluster2",
			URL:               "http://cluster2",
			AppDNS:            "appdns2",
			AuthClientID:      "authclientID2",
			AuthClientSecret:  "authclientsecret2",
			AuthDefaultScope:  "defaultscope2",
			CapacityExhausted: false,
			SAUsername:        "udername2",
			SAToken:           "token2",
			SATokenEncrypted:  true,
			TokenProviderID:   "provider2",
			Type:              "OSO",
		}, result[1])
	})

	t.Run("errors", func(t *testing.T) {

		t.Run("invalid type", func(t *testing.T) {
			// given
			data := []map[string]interface{}{
				{
					"name":               "cluster1",
					"capacity-exhausted": "true", // invalid type, expecting a bool
				},
			}
			// when
			_, err := decodeClusters(data)
			// then
			require.Error(t, err)
		})
	})

}
