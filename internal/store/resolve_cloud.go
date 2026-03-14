//go:build cloud

package store

func init() {
	cloudStoreResolver = func(rootDir, credsPath string) (Store, error) {
		if CloudConfigExists(rootDir) {
			return newCloudStoreFromConfig(rootDir, credsPath)
		}
		return nil, nil
	}
}
