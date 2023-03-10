package routes

import (
	"github.com/gin-gonic/gin"
	"vqlite/config"
	"vqlite/handlers"
)

func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	if config.GlobalConfig.ServiceConfig.RunMode == "debug" {
		gin.SetMode(gin.DebugMode)
		r = gin.Default()
	}

	api := r.Group("/api")
	{
		// health check
		api.GET("/ping", handlers.GetHealth)
		// all vqlite stat
		api.GET("/stat", handlers.VQLiteStat)
		// create colletion
		api.POST("/collection/:target", handlers.CreateCollection)
		// delete collection
		api.DELETE("/collection/:target", handlers.DropCollection)
		// search
		api.POST("/collection/:target/search", handlers.SearchCollection)
		// train
		api.POST("/collection/:target/train", handlers.TrainCollection)
		// dump collection
		api.POST("/collection/:target/dump", handlers.DumpCollection)
		// load collection
		api.POST("/collection/:target/load", handlers.LoadCollection)
		// docs
		api.POST("/collection/:target/document", handlers.AddDocument)
		api.POST("/collection/:target/document/batch", handlers.BatchAddDocuments)
		api.DELETE("/collection/:target/document", handlers.DeleteDocument)
		api.PUT("/collection/:target/document", handlers.UpdateDocumentMetadata)

	}

	r.Use(gin.Recovery())

	return r
}
