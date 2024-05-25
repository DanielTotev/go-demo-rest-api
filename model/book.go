package model

import (
	"books-api/persistence"
	"reflect"
)

type Book struct {
	Id             int    `pk:"true" name:"id" json:"id"`
	Title          string `name:"title" json:"title"`
	Isbn           string `name:"isbn" json:"isbn"`
	Author         string `name:"author" json:"author"`
	PublishingYear int    `name:"publishing_year" json:"publishingYear"`
}

var BookType = reflect.TypeOf(Book{})

func NewBook(title string, isbn string, author string, publishingYear int) Book {
	book := Book{
		Title:          title,
		Isbn:           isbn,
		Author:         author,
		PublishingYear: publishingYear,
	}
	return book
}

func (b *Book) Save() error {
	return persistence.Insert(b)
}

func (b *Book) Update() error {
	return persistence.Update(b)
}

func (b Book) Delete() {
	persistence.DeleteByPrimaryKeyValue(b)
}

func (b *Book) IsEmpty() bool {
	return b.Id == 0 && b.Title == "" && b.Author == "" && b.Isbn == "" && b.PublishingYear == 0
}

func (b *Book) UpdateFields(other Book) {
	b.Author = other.Author
	b.Isbn = other.Isbn
	b.Title = other.Title
	b.PublishingYear = other.PublishingYear
}

func LoadAllBooks() ([]Book, error) {
	return persistence.LoadAll[Book](BookType)
}

func LoadBookById(id int) (Book, error) {
	return persistence.LoadByPrimaryKeyValue[Book](BookType, id)
}

func DeleteBookById(id int) error {
	return persistence.DeleteByPrimaryKeyValue(Book{Id: id})
}
