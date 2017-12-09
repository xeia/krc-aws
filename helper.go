package main

import (
	"errors"
	"github.com/mweagle/Sparta/aws/dynamodb"
	"github.com/xeia/Kings-Raid-Crawler/models"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"time"
	"strings"
	"net/http"
	"encoding/json"
	"os"
	"github.com/xeia/Kings-Raid-Crawler/crawler"
	"bytes"
)

func discordHook(ev dynamodb.Event) error {
	dt := os.Getenv(envDiscordHook)
	if dt == "" {
		return errors.New(envDiscordHookErr)
	}

	var embed models.DiscordEmbed
	embed.Title = "New updates from PLUG cafe"
	embed.Description = "Yoo-hoo! I found new articles published on the PLUG cafe! I'll list them down below for y'all!"
	embed.Color = 3447003
	embed.URL = crawler.CafeBase

	b, _:= json.Marshal(&ev)
	embed.Fields = []models.DiscordField{
		{Name: "Test", Value: string(b)},
	}
	msg := models.DiscordHookMessage{
		Embeds: []models.DiscordEmbed{
			embed,
		},
	}
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	jsonBuf := bytes.NewBuffer(jsonBytes)

	_, err = http.Post(dt, "application/json", jsonBuf)
	if err != nil {
		return err
	}

	return nil
}

func addArticlesToDB(articles []models.Article) ([]models.Article, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess)
	table := db.Table(models.ArticleTable)
	success := true

	// Instead of querying db and building map/getting and checking from db,
	// just do it sequentially
	var results []models.Article
	for _, article := range articles {
		article.CreatedOn = time.Now()
		article.ModifiedOn = time.Now()

		// hackish conditional update to accomodate article revisions
		var oldArticle models.Article
		err := table.Get(models.ArticleIDCol, article.ID).One(&oldArticle)
		if err == dynamo.ErrNotFound {
			err = table.Put(article).If("attribute_not_exists($)", models.ArticleIDCol).Run()
			if err != nil {
				success = false
			} else {
				// may not be needed once streams are done
				results = append(results, article)
			}
		} else if strings.Compare(oldArticle.Title, article.Title) != 0 {
			article.CreatedOn = oldArticle.CreatedOn
			err = table.Put(article).Run()
			if err != nil {
				success = false
			} else {
				results = append(results, article)
			}
		}
	}

	if success {
		return results, nil
	}
	return results, errors.New(dbWriteErr)
}

// isArticleStateUnchanged returns true if the articleHash is the same as
// the currently stored one in the DB for the given articleType
func isArticleStateUnchanged(articleHash string, articleType models.ArticleType, articleId int) bool {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess)
	table := db.Table(models.ArticleStateTable)

	as := models.ArticleState{Type: articleType, ID: articleId, ArticleHash: articleHash}
	var articleState models.ArticleState
	err := table.Get(models.ArticleStateTypeCol, articleType).One(&articleState)
	if err == dynamo.ErrNotFound {
		table.Put(as).Run()
		return false
	}

	if strings.Compare(articleState.ArticleHash, articleHash) == 0 {
		return true
	}
	table.Put(as).Run()
	return false
}

// not needed for now since largest ID will always be the first article returned from the scraper
func getLargestID(articles []models.Article) int {
	v := -1
	for _, article := range articles {
		if article.ID > v {
			v = article.ID
		}
	}
	return v
}

func writeRespHeaderWithMsg(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Write([]byte(message))
}

func writeRespJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
