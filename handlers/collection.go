package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vqlite/core"
)

func GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"remote_ip": c.RemoteIP(),
		"client_ip": c.ClientIP(),
	})
}

func VQLiteStatistics(c *gin.Context) {
	vqliteStatistics := core.Statistics()
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   vqliteStatistics,
	})
}

func CreateCollection(c *gin.Context) {

	var newCol core.CreateCollectionRequest

	if err := c.BindJSON(&newCol); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	collectionName := c.Param("target")
	if collectionName != "" {
		newCol.Name = collectionName
	}
	col, err := core.CreateCollection(newCol.Name, newCol.Dim)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   col.Statistics(),
	})

}

func DropCollection(c *gin.Context) {
	collectionName := c.Param("target")
	core.DropCollection(collectionName)
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func SearchCollection(c *gin.Context) {
	collectionName := c.Param("target")
	var searchReq core.SearchRequest
	if err := c.BindJSON(&searchReq); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := core.SearchCollection(collectionName, searchReq.Vectors, searchReq.Opt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   result,
	})

}
func TrainCollection(c *gin.Context) {
	collectionName := c.Param("target")
	var trainReq core.TrainRequest

	if err := c.BindJSON(&trainReq); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := core.TrainCollection(collectionName, trainReq.Threads, trainReq.IgnoreCheck); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})

}

func DumpCollection(c *gin.Context) {
	collectionName := c.Param("target")

	if err := core.DumpCollection(collectionName); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func DumpCollectionMetadata(c *gin.Context) {
	collectionName := c.Param("target")

	if err := core.DumpCollectionMetadata(collectionName); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
func DumpCollectionIndex(c *gin.Context) {
	collectionName := c.Param("target")

	if err := core.DumpCollectionIndex(collectionName); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func LoadCollection(c *gin.Context) {
	collectionName := c.Param("target")

	if err := core.LoadCollection(collectionName); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})

}
