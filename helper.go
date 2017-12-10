package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	"github.com/mweagle/Sparta/aws/dynamodb"
	"github.com/xeia/Kings-Raid-Crawler/crawler"
	"github.com/xeia/Kings-Raid-Crawler/models"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func discordHook(ev dynamodb.Event, logger *logrus.Logger) error {
	var embeds []models.DiscordEmbed
	for _, rec := range ev.Records {
		e, err := parseSingleRecord(rec)
		if err != nil {
			logger.Error(err)
		}
		embeds = append(embeds, e)
	}

	msg := models.DiscordHookMessage{
		Content: generateContentString(),
		Embeds:  embeds,
	}

	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return sendDiscordHook(jsonBytes)
}

func sendDiscordHook(b []byte) error {
	dt := os.Getenv(envDiscordHook)
	if dt == "" {
		return errors.New(envDiscordHookErr)
	}
	return sendHook(dt, b)
}

func sendHook(url string, b []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK || resp.StatusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("Response code received: %d", resp.StatusCode))
	}
	return nil
}

func parseSingleRecord(rec dynamodb.EventRecord) (models.DiscordEmbed, error) {
	var embed models.DiscordEmbed
	embed.Title = *rec.DynamoDB.NewImage["article-title"].S
	embed.Description = *rec.DynamoDB.NewImage["article-description"].S
	embed.Thumbnail = models.DiscordThumbnail{URL: *rec.DynamoDB.NewImage["article-thumb-url"].S}

	articleID, err := strconv.Atoi(*rec.DynamoDB.NewImage["article-id"].N)
	if err != nil {
		return embed, err
	}

	articleType, err := strconv.Atoi(*rec.DynamoDB.NewImage["article-type"].N)
	if err != nil {
		return embed, err
	}

	embed.URL = formatArticleURL(models.ArticleType(articleType), articleID)
	embed.Color = generateColorCode(models.ArticleType(articleType))
	return embed, nil
}

func formatArticleURL(at models.ArticleType, id int) string {
	switch at {
	case models.EVENTS:
		return fmt.Sprintf(crawler.ShowEventFormat, id)
	case models.NOTICE:
		return fmt.Sprintf(crawler.ShowNoticeFormat, id)
	case models.PATCHNOTES:
		return fmt.Sprintf(crawler.ShowPatchNoteFormat, id)
	}
	return ""
}

// generateColorCode generates color codes in decimal for different category
func generateColorCode(at models.ArticleType) int {
	switch at {
	case models.EVENTS:
		return 3447003
	case models.NOTICE:
		return 14382900
	case models.PATCHNOTES:
		return 3464055
	}
	return 14365765
}

// generateContentString returns a randomized string to be used in the discord message
func generateContentString() string {
	s := []string{
		"Yoo-hoo! I found a new update on the PLUG cafe! I'll list them here for y'all!",
		"Yippe! New updates from the PLUG cafe!",
	}
	return s[rand.Intn(len(s))]
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
		if strings.Compare(err.Error(), dynamo.ErrNotFound.Error()) == 0 {
			err = table.Put(article).If("attribute_not_exists(article-id)").Run()
			if err != nil {
				success = false
			} else {
				// may not be needed once streams are done
				results = append(results, article)
			}
		} else if err == nil && strings.Compare(oldArticle.Title, article.Title) != 0 {
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

func getArticlesFromDB() ([]models.Article, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess)

	var res []models.Article
	aTable := db.Table(models.ArticleTable)
	err := aTable.Scan().All(&res)
	return res, err
}

func getLatestArticleByTypeFromDB(at models.ArticleType, limit int64) ([]models.Article, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess)

	aTable := db.Table(models.ArticleTable)
	asTable := db.Table(models.ArticleStateTable)
	var as models.ArticleState
	var res []models.Article

	err := asTable.Get(models.ArticleStateTypeCol, at).One(&as)
	if err != nil {
		return res, err
	}

	err = aTable.Get(models.ArticleIDCol, as.ID).Limit(limit).All(&res)
	return res, err
}

func getLatestArticleFromDB() (models.Article, error) {
	sess := session.Must(session.NewSession())
	db := dynamo.New(sess)

	aTable := db.Table(models.ArticleTable)
	asTable := db.Table(models.ArticleStateTable)
	var all []models.ArticleState
	var a models.Article

	err := asTable.Scan().All(&all)
	if err != nil {
		return a, err
	}

	max := -1
	for _, as := range all {
		if as.ID > max {
			max = as.ID
		}
	}

	err = aTable.Get(models.ArticleIDCol, max).One(&a)
	return a, err
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

func convertURLReqType(a string) (models.ArticleType, error) {
	switch strings.ToLower(a) {
	case "event":
	case "events":
		return models.EVENTS, nil
	case "notice":
	case "notices":
		return models.NOTICE, nil
	case "patch":
	case "patchnotes":
	case "patch_notes":
	case "patch-notes":
		return models.PATCHNOTES, nil
	}
	return models.NOTICE, errors.New("No known article-type found")
}

func writeRespHeaderWithMsg(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Write([]byte(message))
}

func writeRespJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}
