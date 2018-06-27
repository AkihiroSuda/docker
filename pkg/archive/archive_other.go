// +build !linux

package archive // import "github.com/docker/docker/pkg/archive"

func getWhiteoutConverter(format WhiteoutFormat, bool inUserNS) tarWhiteoutConverter {
	return nil
}
