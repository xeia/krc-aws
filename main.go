package main

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mweagle/Sparta"
	"github.com/mweagle/Sparta/aws/dynamodb"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/xeia/Kings-Raid-Crawler/crawler"
	"github.com/xeia/Kings-Raid-Crawler/models"
	"net/http"
	"os"
	"strconv"
)

const (
	envDynamoDBStream    = "DYNAMO_DBSTREAM"
	envDynamoDBStreamErr = "env DYNAMO_DBSTREAM does not exist"

	envDiscordHook    = "DISCORD_WEBHOOK"
	envDiscordHookErr = "env DISCORD_WEBHOOK does not exist"
	enableDiscordHook = true

	envTelegram    = "TELEGRAM_TOKEN"
	envTelegramErr = "env TELEGRAM_TOKEN does not exist"
	enableTelegram = false
)

const (
	dbWriteErr   = "error writing to db"
	dbReadErr    = "error reading from db"
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
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, eventReadErr+err.Error())
	}

	for _, rec := range lambdaEvent.Records {
		logger.WithFields(logrus.Fields{
			"NewImage": rec.DynamoDB.NewImage,
		}).Info("DynamoDB event")
	}

	if enableDiscordHook {
		err = discordHook(lambdaEvent, logger)
		if err != nil {
			logger.Error("DiscordHook Error :", err.Error())
		} else {
			logger.Info("DiscordHook successfully sent!")
		}
	}

	if enableTelegram {
		// todo
	}

	writeRespHeaderWithMsg(w, http.StatusNoContent, "")
}

func queryAll(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	res, err := getArticlesFromDB()
	if err != nil {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	}

	b, err := json.Marshal(&res)
	if err != nil {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	}

	writeRespJSON(w, b)
}

func queryByType(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	var res []models.Article
	t := r.URL.Query().Get("type")
	limit := r.URL.Query().Get("limit")

	at, err := convertURLReqType(t)
	if err != nil || len(t) == 0 {
		writeRespHeaderWithMsg(w, http.StatusBadRequest, "Missing or unknown article type found in request")
	}

	if len(limit) != 0 {
		l, err := strconv.ParseInt(limit, 10, 64)
		if err != nil {
			writeRespHeaderWithMsg(w, http.StatusBadRequest, err.Error())
		}

		res, err = getLatestArticleByTypeFromDB(at, l)
		if err != nil {
			writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
		}
	} else {
		res, err = getLatestArticleByTypeFromDB(at, 1)
		if err != nil {
			writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
		}
	}

	b, err := json.Marshal(&res)
	if err != nil {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	}
	writeRespJSON(w, b)
}

func queryLatest(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(sparta.ContextKeyLambdaContext).(*sparta.LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info(requestReceived)

	res, err := getLatestArticleFromDB()
	if err != nil {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	}

	b, err := json.Marshal(&res)
	if err != nil {
		writeRespHeaderWithMsg(w, http.StatusInternalServerError, err.Error())
	}
	writeRespJSON(w, b)
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
				"ArticleID":    article.ID,
				"ArticleTitle": article.Title,
			}).Info("ScrapeAll Events")
		}
	}

	noticeArticles, noticeHash := crawler.ScrapeNotices()
	if len(noticeArticles) > 0 && !isArticleStateUnchanged(noticeHash, models.NOTICE, noticeArticles[0].ID) {
		result = append(result, noticeArticles...)
		for _, article := range noticeArticles {
			logger.WithFields(logrus.Fields{
				"ArticleID":    article.ID,
				"ArticleTitle": article.Title,
			}).Info("ScrapeAll Notices")
		}
	}

	patchNotesArticles, patchNotesHash := crawler.ScrapePatchNotes()
	if len(patchNotesArticles) > 0 && !isArticleStateUnchanged(patchNotesHash, models.PATCHNOTES, patchNotesArticles[0].ID) {
		result = append(result, patchNotesArticles...)
		for _, article := range patchNotesArticles {
			logger.WithFields(logrus.Fields{
				"ArticleID":    article.ID,
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

func createLambdaOptions(desc string, timeout int64, env map[string]*gocf.StringExpr) *sparta.LambdaFunctionOptions {
	return &sparta.LambdaFunctionOptions{
		Description: desc,
		MemorySize:  128,
		Timeout:     timeout,
		Environment: env,
	}
}

func spartaLambdaFunctions(api *sparta.API) []*sparta.LambdaAWSInfo {
	var lambdaFunctions []*sparta.LambdaAWSInfo
	envMap := make(map[string]*gocf.StringExpr)
	envMap[envDynamoDBStream] = gocf.String(os.Getenv(envDynamoDBStream))
	envMap[envDiscordHook] = gocf.String(os.Getenv(envDiscordHook))
	envMap[envTelegram] = gocf.String(os.Getenv(envTelegram))

	scrapeAllFn := sparta.HandleAWSLambda("Scrape All", http.HandlerFunc(scrapeAll), sparta.IAMRoleDefinition{})
	scrapeAllFn.Options = createLambdaOptions("Scrapes PLUG cafe for notices/events/patch notes", 270, envMap)
	lambdaFunctions = append(lambdaFunctions, scrapeAllFn)

	scrapeEventsFn := sparta.HandleAWSLambda("Scrape Events", http.HandlerFunc(scrapeEvents), sparta.IAMRoleDefinition{})
	scrapeEventsFn.Options = createLambdaOptions("Scrapes PLUG cafe for events", 150, envMap)
	lambdaFunctions = append(lambdaFunctions, scrapeEventsFn)

	scrapeNoticesFn := sparta.HandleAWSLambda("Scrape Notices", http.HandlerFunc(scrapeNotices), sparta.IAMRoleDefinition{})
	scrapeNoticesFn.Options = createLambdaOptions("Scrapes PLUG cafe for notices", 150, envMap)
	lambdaFunctions = append(lambdaFunctions, scrapeNoticesFn)

	scrapePatchNotesFn := sparta.HandleAWSLambda("Scrape Patch Notes", http.HandlerFunc(scrapePatchNotes), sparta.IAMRoleDefinition{})
	scrapePatchNotesFn.Options = createLambdaOptions("Scrapes PLUG cafe for patch notes", 150, envMap)
	lambdaFunctions = append(lambdaFunctions, scrapePatchNotesFn)

	handleArticleFn := sparta.HandleAWSLambda("Handle New Articles", http.HandlerFunc(handleNewArticles), sparta.IAMRoleDefinition{})
	handleArticleFn.Options = createLambdaOptions("Handles updates from DB stream to be published", 150, envMap)
	dbStream := os.Getenv(envDynamoDBStream)
	if dbStream == "" {
		panic(envDynamoDBStreamErr)
	}
	handleArticleFn.EventSourceMappings = append(handleArticleFn.EventSourceMappings,
		&sparta.EventSourceMapping{
			EventSourceArn:   dbStream,
			StartingPosition: "TRIM_HORIZON",
			BatchSize:        10,
		})
	lambdaFunctions = append(lambdaFunctions, handleArticleFn)

	queryAllFn := sparta.HandleAWSLambda("Query All", http.HandlerFunc(queryAll), sparta.IAMRoleDefinition{})
	queryAllFn.Options = createLambdaOptions("Queries the database to retrieve all articles", 30, envMap)
	lambdaFunctions = append(lambdaFunctions, queryAllFn)

	queryByTypeFn := sparta.HandleAWSLambda("Query By Type", http.HandlerFunc(queryAll), sparta.IAMRoleDefinition{})
	queryByTypeFn.Options = createLambdaOptions("Queries the database to retrieve articles by type", 30, envMap)
	lambdaFunctions = append(lambdaFunctions, queryByTypeFn)

	queryLatestFn := sparta.HandleAWSLambda("Query Latest", http.HandlerFunc(queryAll), sparta.IAMRoleDefinition{})
	queryLatestFn.Options = createLambdaOptions("Queries the database to retrieve the latest article", 10, envMap)
	lambdaFunctions = append(lambdaFunctions, queryLatestFn)

	if api != nil {
		scrapeAllRes, _ := api.NewResource("/scrape", scrapeAllFn)
		_, err := scrapeAllRes.NewMethod(http.MethodPost, http.StatusOK)
		if err != nil {
			panic("Failed to create /scrape resource")
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

		queryAllRes, _ := api.NewResource("/get/all", queryAllFn)
		_, err = queryAllRes.NewMethod(http.MethodGet, http.StatusOK)
		if err != nil {
			panic("Failed to create /get/all resource")
		}

		queryLatestRes, _ := api.NewResource("/get/latest", queryLatestFn)
		_, err = queryLatestRes.NewMethod(http.MethodGet, http.StatusOK)
		if err != nil {
			panic("Failed to create /get/latest resource")
		}

		queryByTypeRes, _ := api.NewResource("/get", queryByTypeFn)
		_, err = queryByTypeRes.NewMethod(http.MethodGet, http.StatusOK)
		if err != nil {
			panic("Failed to create /get resource")
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
