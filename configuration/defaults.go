package configuration

const (
	devModePublicKey = `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvQ8p+HsTMrgcsuIMoOR1
LXRhynL9YAU0qoDON6PLKCpdBv0Xy/jnsPjo5DrtUOijuJcID8CR7E0hYpY9MgK5
H5pDFwC4lbUVENquHEVS/E0pQSKCIzSmORcIhjYW2+wKfDOVjeudZwdFBIxJ6KpI
ty/aF78hlUJZuvghFVqoHQYTq/DZOmKjS+PAVLw8FKE3wa/3WU0EkpP+iovRMCkl
lzxqrcLPIvx+T2gkwe0bn0kTvdMOhTLTN2tuvKrFpVUxVi8RM/V8PtgdKroxnES7
SyUqK8rLO830jKJzAYrByQL+sdGuSqInIY/geahQHEGTwMI0CLj6zfhpjSgCflst
vwIDAQAB
-----END RSA PUBLIC KEY-----`

	devModePublicKeyID = "bNq-BCOR3ev-E6buGSaPrU-0SXX8whhDlmZ6geenkTE"

	defaultDBPassword = "mysecretpassword"

	defaultLogLevel = "info"

	osoClusterConfigFileName    = "oso-clusters.conf"
	defaultOsoClusterConfigPath = "/etc/fabric8/" + osoClusterConfigFileName

	prodEnvironment        = "production"
	prodPreviewEnvironment = "prod-preview"
)
