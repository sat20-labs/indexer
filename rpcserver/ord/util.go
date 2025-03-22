package ord

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/sat20-labs/indexer/common"
)

func genContentHeader(c *gin.Context, nft *common.Nft) (string, error) {
	acceptEncoding := c.GetHeader("accept-encoding")
	var acceptEncodingList []string
	if acceptEncoding != "" {
		parts := strings.Split(acceptEncoding, ",")
		acceptEncodingList = make([]string, len(parts))
		for i, part := range parts {
			subparts := strings.Split(part, ";")
			acceptEncodingList[i] = strings.TrimSpace(subparts[0])
		}
	}

	contentEncoding, err := strconv.Unquote(fmt.Sprintf("%q", nft.Base.ContentEncoding))
	if err != nil {
		return "", fmt.Errorf("failed to unquote %q, error: %w", nft.Base.ContentEncoding, err)
	}

	if contentEncoding != "" && len(acceptEncodingList) > 0 {
		findAcceptable := false
		for _, name := range acceptEncodingList {
			if name == contentEncoding {
				c.Writer.Header().Set("content-encoding", contentEncoding)
				findAcceptable = true
				break
			}
		}
		if !findAcceptable {
			return "", fmt.Errorf("content encoding %q not acceptable, acceptEncoding: %q", contentEncoding, acceptEncoding)
		}
	}

	c.Writer.Header().Set(
		CONTENT_SECURITY_POLICY,
		"default-src 'self' 'unsafe-eval' 'unsafe-inline' data: blob:",
	)
	c.Writer.Header().Add(
		CONTENT_SECURITY_POLICY,
		"default-src *:*/content/ *:*/blockheight *:*/blockhash *:*/blockhash/ *:*/blocktime *:*/r/ 'unsafe-eval' 'unsafe-inline' data: blob:",
	)
	c.Writer.Header().Set(
		CACHE_CONTROL,
		"public, max-age=1209600, immutable",
	)

	contentType := string(nft.Base.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return contentType, nil
}

func getMediaType(nft *common.Nft) MediaType {
	if len(nft.Base.Content) == 0 {
		return Unknown
	}
	mediaInfo, ok := MediaList[string(nft.Base.ContentType)]
	if !ok {
		return Unknown
	}
	return mediaInfo.Type
}

func checkInscriptionId(inscriptionId string) error {
	if len(inscriptionId) < MIN_INSCRIPTIONID_LEN {
		return fmt.Errorf("invalid URL: invalid length: %d", len(inscriptionId))
	}
	for _, char := range inscriptionId {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return fmt.Errorf(`invalid URL: invalid character: %c`, char)
		}
	}
	separator := inscriptionId[TXID_LEN]
	if separator != 'i' {
		return fmt.Errorf(`invalid URL: invalid separator: %c`, separator)
	}
	return nil
}
