package routes

import (
	"books-api/model"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func SetBookRoutes(router *gin.Engine) {
	router.GET("/books", getAllBooks)
	router.GET("/books/:book_id", getBookById)
	router.POST("/books", createBook)
	router.PUT("/books/:book_id", updateBook)
	router.DELETE("/books/:book_id", deleteBook)
}

func getAllBooks(ctx *gin.Context) {
	var books, err = model.LoadAllBooks()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": INTERNAL_SERVER_ERROR_MESSAGE})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"books": books})
}

func getBookById(ctx *gin.Context) {
	book, hasError := loadBook(ctx)
	if hasError {
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"book": book})
}

func createBook(ctx *gin.Context) {
	var book model.Book
	parseBodyErr := ctx.ShouldBindJSON(&book)
	if parseBodyErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": INVALID_REQUEST_BODY_ERROR_MESSAGE})
		return
	}
	saveError := book.Save()
	if saveError != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": INTERNAL_SERVER_ERROR_MESSAGE})
	}
	ctx.JSON(http.StatusCreated, gin.H{"book": book})
}

func updateBook(ctx *gin.Context) {
	var requestBody model.Book
	parseBodyErr := ctx.ShouldBindJSON(&requestBody)
	if parseBodyErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": INVALID_REQUEST_BODY_ERROR_MESSAGE})
		return
	}

	book, errorLoadingBook := loadBook(ctx)
	if errorLoadingBook {
		return
	}

	book.UpdateFields(requestBody)

	updateError := book.Update()
	if updateError != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": INTERNAL_SERVER_ERROR_MESSAGE})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"book": book})
}

func deleteBook(ctx *gin.Context) {
	bookId, parseFailed := parseBookId(ctx)
	if parseFailed {
		return
	}
	deleteErr := model.DeleteBookById(bookId)
	if deleteErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": INTERNAL_SERVER_ERROR_MESSAGE})
		return
	}
	ctx.Status(http.StatusNoContent)
}

func parseBookId(ctx *gin.Context) (int, bool) {
	bookIdString := ctx.Param("book_id")
	bookId, err := strconv.Atoi(bookIdString)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": INVALID_BOOK_ID_ERROR_MESSAGE})
		return 0, true
	}
	return bookId, false
}

func loadBook(ctx *gin.Context) (model.Book, bool) {
	bookId, parseFailed := parseBookId(ctx)
	if parseFailed {
		return model.Book{}, true
	}
	book, loadBokErr := model.LoadBookById(bookId)
	if loadBokErr != nil {
		fmt.Println(loadBokErr.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": INTERNAL_SERVER_ERROR_MESSAGE})
		return model.Book{}, true
	}

	if book.IsEmpty() {
		ctx.JSON(http.StatusNotFound, gin.H{"error": NOT_FOUND_ERROR_MESSAGE})
		return book, true
	}
	return book, false
}
