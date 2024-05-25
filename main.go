package main

import (
	"books-api/model"
	"books-api/persistence"
	"books-api/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	persistence.Init()
	persistence.CreateTable(model.BookType)
	router := gin.Default()
	routes.SetBookRoutes(router)
	router.Run() // listen and serve on 0.0.0.0:8080
	db := persistence.GetDb()
	defer db.Close()
}
