package routes

import (
	"github.com/chenjiandongx/ginprom"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"vqlite/config"
	"vqlite/core"
	"vqlite/handlers"
)

func InitRouter() *gin.Engine {
	// load all collections
	core.LoadAllCollections()

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(ginprom.PromMiddleware(nil))

	if config.GlobalConfig.ServiceConfig.RunMode == "debug" {
		gin.SetMode(gin.DebugMode)
		r = gin.Default()
	}

	// register the `/metrics` route.
	r.GET("/metrics", ginprom.PromHandler(promhttp.Handler()))

	api := r.Group("/api")
	{
		// health check
		api.GET("/ping", handlers.GetHealth)
		// all vqlite stat
		api.GET("/stat", handlers.VQLiteStatistics)
		api.GET("/statistics", handlers.VQLiteStatistics)
		// create colletion
		api.POST("/collection/:target", handlers.CreateCollection)
		// delete collection
		api.DELETE("/collection/:target", handlers.DropCollection)
		// search
		api.POST("/collection/:target/search", handlers.SearchCollection)
		// train
		api.POST("/collection/:target/train", handlers.TrainCollection)

		// dump collection
		//api.POST("/collection/:target/dump", handlers.DumpCollection)
		api.POST("/collection/:target/dump", handlers.DumpCollection)
		api.POST("/collection/:target/dump/metadata", handlers.DumpCollectionMetadata)
		api.POST("/collection/:target/dump/index", handlers.DumpCollectionIndex)
		// load collection
		api.POST("/collection/:target/load", handlers.LoadCollection)

		// docs
		api.POST("/collection/:target/document", handlers.AddDocument)
		api.POST("/collection/:target/document/batch", handlers.BatchAddDocuments)
		api.DELETE("/collection/:target/document", handlers.DeleteDocument)
		api.PUT("/collection/:target/document", handlers.UpdateDocumentMetadata)
		api.GET("/collection/:target/document", handlers.GetDocumentMetadata)
	}

	pprof.Register(r)
	r.Use(gin.Recovery())

	return r
}
