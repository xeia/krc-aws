package main

import (
	"github.com/mweagle/Sparta"
	"github.com/mweagle/Sparta/aws/dynamodb"
	"github.com/xeia/Kings-Raid-Crawler/crawler"
	"github.com/xeia/Kings-Raid-Crawler/models"
	"net/http"
	"encoding/json"
	"github.com/Sirupsen/logrus"
)

const (
	envDynamoDB = "DYNAMO_DB"
	envDynamoDBErr = "Env DYNAMO_DB does not exist"

	envDiscordHook    = "DISCORD_WEBHOOK"
	envDiscordHookErr = "Env DISCORD_WEBHOOK does not exist"
	enableDiscordHook = true

	envTelegram = "TELEGRAM_TOKEN"
	envTelegramErr = "Env TELEGRAM_TOKEN does not exist"
	enableTelegram = false
)

const (
	dbWriteErr = "error writing to db"
	dbReadErr  = "error reading from db"
	eventReadErr = "error unmarshalling event data: "

	requestReceived = "Request received"

	scrapeComplete = "Scraping completed successfully!"
	stateUnchanged = "No new articles have been published at this time. Please check back again later."
)

func handleNewArticles(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var lambdaEvent dynamodb.Event
	err := decoder.Decode(&lambdaEvent)
	if err != nil {
		logger.Error(eventReadErr, err.Error())
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, eventReadErr + err.Error())
	}

	for _, rec := range lambdaEvent.Records {
		logger.WithFields(logrus.Fields{
			"Keys": rec.DynamoDB.Keys,
			"NewImage": rec.DynamoDB.NewImage,
		}).Info("DynamoDB event")
	}

	if enableDiscordHook {
		err := discordHook(lambdaEvent)
		if err != nil {
			logger.Error("DiscordHook Error :", err.Error())
		}
	}

	if enableTelegram {
		// todo
	}

	bytes, bytesErr := json.Marshal(&lambdaEvent)
	if bytes != nil {
		writeRespJSON(w, bytes)
	} else {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, bytesErr.Error())
	}
}

func scrapeAll(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	var result []models.Article

	eventsArticles, eventsHash := crawler.ScrapeEvents()
	if len(eventsArticles) > 0 && !isArticleStateUnchanged(eventsHash, models.EVENTS, eventsArticles[0].ID) {
		result = append(result, eventsArticles...)
		for _, article := range eventsArticles {
			logger.WithFields(logrus.Fields{
				"ArticleID": article.ID,
				"ArticleTitle": article.Title,
			}).Info("ScrapeAll Events")
		}
	}

	noticeArticles, noticeHash := crawler.ScrapeNotices()
	if len(noticeArticles) > 0 && !isArticleStateUnchanged(noticeHash, models.NOTICE, noticeArticles[0].ID) {
		result = append(result, noticeArticles...)
		for _, article := range noticeArticles {
			logger.WithFields(logrus.Fields{
				"ArticleID": article.ID,
				"ArticleTitle": article.Title,
			}).Info("ScrapeAll Notices")
		}
	}

	patchNotesArticles, patchNotesHash := crawler.ScrapePatchNotes()
	if len(patchNotesArticles) > 0 && !isArticleStateUnchanged(patchNotesHash, models.PATCHNOTES, patchNotesArticles[0].ID) {
		result = append(result, patchNotesArticles...)
		for _, article := range patchNotesArticles {
			logger.WithFields(logrus.Fields{
				"ArticleID": article.ID,
				"ArticleTitle": article.Title,
			}).Info("ScrapeAll Patch Notes")
		}
	}

	if len(result) < 1 {
		logger.Info("ScrapeAll Unchanged")
		writeRespHeaderWithMsg(w, http.StatusNotModified, stateUnchanged)
		return
	}
	_, err := addArticlesToDB(result)
	if err != nil {
		logger.Error("ScrapeAll Error Add", err.Error())
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	} else {
		logger.Info("ScrapeAll Complete")
		writeRespHeaderWithMsg(w, http.StatusOK, scrapeComplete)
	}
}

func scrapeEvents(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	articles, articleHash := crawler.ScrapeEvents()
	if len(articles) < 1 || (len(articles) > 0 && isArticleStateUnchanged(articleHash, models.EVENTS, articles[0].ID)) {
		logger.Info("ScrapeEvents Unchanged")
		writeRespHeaderWithMsg(w, http.StatusNotModified, stateUnchanged)
		return
	}

	_, err := addArticlesToDB(articles)
	if err != nil {
		logger.Error("ScrapeEvents Error Add", err.Error())
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	} else {
		logger.Info("ScrapeEvents Complete")
		writeRespHeaderWithMsg(w, http.StatusOK, scrapeComplete)
	}
}

func scrapeNotices(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	articles, articleHash := crawler.ScrapeNotices()
	if len(articles) < 1 || (len(articles) > 0 && isArticleStateUnchanged(articleHash, models.NOTICE, articles[0].ID)) {
		logger.Info("ScrapeNotices Unchanged")
		writeRespHeaderWithMsg(w, http.StatusNotModified, stateUnchanged)
		return
	}

	_, err := addArticlesToDB(articles)
	if err != nil {
		logger.Error("ScrapeNotices Error Add", err.Error())
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	} else {
		logger.Info("ScrapeNotices Complete")
		writeRespHeaderWithMsg(w, http.StatusOK, scrapeComplete)
	}
}

func scrapePatchNotes(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	articles, articleHash := crawler.ScrapePatchNotes()
	if len(articles) < 1 || (len(articles) > 0 && isArticleStateUnchanged(articleHash, models.PATCHNOTES, articles[0].ID)) {
		logger.Info("ScrapePatchNotes Unchanged")
		writeRespHeaderWithMsg(w, http.StatusNotModified, stateUnchanged)
		return
	}

	_, err := addArticlesToDB(articles)
	if err != nil {
		logger.Error("ScrapePatchNotes Error Add", err.Error())
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	} else {
		logger.Info("ScrapePatchNotes Complete")
		writeRespHeaderWithMsg(w, http.StatusOK, scrapeComplete)
	}
}

func spartaLambdaFunctions(api *sparta.API) []*sparta.LambdaAWSInfo {
	var lambdaFunctions []*sparta.LambdaAWSInfo

	scrapeAllFn := sparta.HandleAWSLambda("Scrape All", http.HandlerFunc(scrapeAll), sparta.IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, scrapeAllFn)

	scrapeEventsFn := sparta.HandleAWSLambda("Scrape Events", http.HandlerFunc(scrapeEvents), sparta.IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, scrapeEventsFn)

	scrapeNoticesFn := sparta.HandleAWSLambda("Scrape Notices", http.HandlerFunc(scrapeNotices), sparta.IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, scrapeNoticesFn)

	scrapePatchNotesFn := sparta.HandleAWSLambda("Scrape Patch Notes", http.HandlerFunc(scrapePatchNotes), sparta.IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, scrapePatchNotesFn)

	if api != nil {
		scrapeAllRes, _ := api.NewResource("/scrape/all", scrapeAllFn)
		_, err := scrapeAllRes.NewMethod(http.MethodPost, http.StatusOK)
		if err != nil {
			panic("Failed to create /scrape/all resource")
		}

		scrapeEventRes, _ := api.NewResource("/scrape/events", scrapeEventsFn)
		_, err = scrapeEventRes.NewMethod(http.MethodPost, http.StatusOK)
		if err != nil {
			panic("Failed to create /scrape/events resource")
		}

		scrapeNoticeRes, _ := api.NewResource("/scrape/notices", scrapeNoticesFn)
		_, err = scrapeNoticeRes.NewMethod(http.MethodPost, http.StatusOK)
		if err != nil {
			panic("Failed to create /scrape/notices resource")
		}

		scrapePatchRes, _ := api.NewResource("/scrape/patch", scrapePatchNotesFn)
		_, err = scrapePatchRes.NewMethod(http.MethodPost, http.StatusOK)
		if err != nil {
			panic("Failed to create /scrape/patch resource")
		}
	}

	return lambdaFunctions
}

func main() {
	apiStage := sparta.NewStage("v1")
	apiGateway := sparta.NewAPIGateway("KingsRaidCrawler", apiStage)

	sparta.Main("KingsRaidCrawlerStack",
		"Kings Raid Crawler Core Functionality",
		spartaLambdaFunctions(apiGateway),
		apiGateway,
		nil)
}
