package models

import "time"

// ArticleType represents a category of the published Article
type ArticleType int

// Enum for ArticleType
const (
	NOTICE ArticleType = 1 + iota
	EVENTS
	PATCHNOTES
)

// Article Table columns
const (
	ArticleTable     = "kr-articles"
	ArticleTypeCol   = "article-type"
	ArticleIDCol     = "article-id"
	ArticleTitleCol  = "title"
	ArticleDescCol   = "description"
	ArticleImgURLCol = "thumb-url"
)

// Article representing a published article on PLUG Cafe
type Article struct {
	Type       ArticleType `dynamo:article-type` // primary partition key
	ID         int         `dynamo:article-id`   // primary sort key
	Title      string      `dynamo:title`
	Desc       string      `dynamo:description`
	ImgURL     string      `dynamo:thumb-url`
	CreatedOn  time.Time   `dynamo:created-on`
	ModifiedOn time.Time   `dynamo:modified-on`
}
