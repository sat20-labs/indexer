package ord

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticRes embed.FS

type Service struct {
	//handle *Handle
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) InitRouter(r *gin.Engine, basePath string) {
	// static resource
	fileServer := http.FileServer(http.FS(staticRes))
	r.GET("/static/*filepath", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	// ordinal status
	g := r.Group(basePath + "/ord")
	// g.GET("/status", s.getOrdinalStatus)
	// the content of the inscription with <INSCRIPTION_ID>, allow cached
	g.GET("/content/:inscriptionid", s.getInscriptionContent)
	// the preview of the inscription with <INSCRIPTION_ID>, allow cached
	g.GET("/preview/:inscriptionid", s.getInscriptionPreview)
	// ord recursive endpoints
	// block hash at given block height, allow cached
	// g.GET("/r/blockhash/:height", s.getRBlockHash)
	// latest block hash, no allow cached
	// g.GET("/r/blockhash", s.getRLastestBlockHash)
	// latest block height, no allow cached
	// g.GET("/r/blockheight", s.getRLastestBlockHeight)
	// block info, <QUERY> may be a block height or block hash, allow cached
	// g.GET("/r/blockinfo/:query", s.getRBlockInfo)
	// UNIX time stamp of latest block, no allow cached
	// g.GET("/r/blocktime", s.getRLatestBlockTimestamp)
	// the first 100 child inscription ids, no allow cached?
	// g.GET("/r/children/:inscriptionid", s.getRChildrenInscriptionIdList)
	// the set of 100 child inscription ids on <PAGE>, no allow cached?
	// g.GET("/r/children/:inscriptionid/:page", s.getRChildrenInscriptionIdList)
	// information about an inscription, allow cached
	// g.GET("/r/inscription/:inscriptionid", s.getRInscriptionInfo)
	// JSON string containing the hex-encoded CBOR metadata, allow cached
	g.GET("/r/metadata/:inscriptionid", s.getRMetadata)
	// the first 100 inscription ids on a sat, no allow cached?
	// g.GET("/r/sat/:satnumber", s.getRSatInscriptionIdList)
	// the set of 100 inscription ids on <PAGE>, no allow cached?
	// g.GET("/r/sat/:satnumber/:page", s.getRSatInscriptionIdList)
	// the inscription id at <INDEX> of all inscriptions on a sat, allow cached
	// <INDEX> may be a negative number to index from the back. 0 being the first and -1 being the most recent for example.
	// g.GET("/r/sat/:satnumber/at/:index", s.getRSatInscriptionId)
}
