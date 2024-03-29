package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vqlite/core"
)

func AddDocument(c *gin.Context) {
	collectionName := c.Param("target")

	var doc core.AddDocumentRequest

	if err := c.BindJSON(&doc); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := core.AddDocument(collectionName, &doc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})

}

func BatchAddDocuments(c *gin.Context) {
	collectionName := c.Param("target")

	var docs core.BatchAddDocumentsRequest
	if err := c.BindJSON(&docs); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(docs.Documents) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Documents is empty"})
		return
	}
	err := core.BatchAddDocuments(collectionName, &docs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func DeleteDocument(c *gin.Context) {
	collectionName := c.Param("target")
	var doc core.DeleteDocumentRequest
	if err := c.BindJSON(&doc); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vqid := doc.Vqid
	deletedCount, err := core.DeleteDocument(collectionName, vqid)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"count":  deletedCount,
	})
}

func UpdateDocumentMetadata(c *gin.Context) {
	collectionName := c.Param("target")
	var doc core.UpdateDocumentMetadataRequest
	if err := c.BindJSON(&doc); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedCount, err := core.UpdateDocumentMetadata(collectionName, &doc)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"count":  updatedCount,
	})
}

func GetDocumentMetadata(c *gin.Context) {
	collectionName := c.Param("target")
	vqid := c.Query("vqid")
	all := c.Query("all")

	checkDuplicate := false
	switch all {
	case "true":
		checkDuplicate = true
	case "1":
		checkDuplicate = true
	case "True":
		checkDuplicate = true
	default:
		checkDuplicate = false
	}
	metadataList, err := core.GetDocumentMetadata(collectionName, vqid, checkDuplicate)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   metadataList,
	})

}
