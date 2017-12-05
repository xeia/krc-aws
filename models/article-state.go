package models

// ArticleState table const
const (
	ArticleStateTable   = "kr-article-state"
	ArticleStateTypeCol = "article-type"
	ArticleStateIDCol   = "article-id"
	ArticleStateHashCol = "article-hash"
)

// ArticleState represents the latest poll of
// each category's article on PLUG Cafe
type ArticleState struct {
	Type        ArticleType `dynamo:"article-type"` // primary partition key
	ID          int         `dynamo:"article-id"`   // primary sort key
	ArticleHash string      `dynamo:"article-hash"`
}
