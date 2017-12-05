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

// Article table const
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
	ID         int         `dynamo:"article-id",json:"article_id"`     // primary partition key
	Type       ArticleType `dynamo:"article-type",json:"article_type"` // primary sort key
	Title      string      `dynamo:"article-title",json:"article_title"`
	Desc       string      `dynamo:"article-description",json:"article_description"`
	ImgURL     string      `dynamo:"article-thumb-url",json:"article_thumb_url"`
	CreatedOn  time.Time   `dynamo:"created-on",json:"-"`
	ModifiedOn time.Time   `dynamo:"modified-on",json:"-"`
}
