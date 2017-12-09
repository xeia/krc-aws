package crawler

import (
	"crypto/sha1"
	"encoding/hex"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"github.com/xeia/Kings-Raid-Crawler/models"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// PLUG cafe links
const (
	CafeBase            = "https://www.plug.game/kingsraid-en"
	ShowNoticeFormat    = noticesUrl + "/posts/%d"
	ShowEventFormat     = eventsUrl + "/posts/%d"
	ShowPatchNoteFormat = patchNotesUrl + "/posts/%d"
)

const (
	noticesUrl    = CafeBase + "/posts?menuId=1#"
	eventsUrl     = CafeBase + "/posts?menuId=2#"
	patchNotesUrl = CafeBase + "/posts?menuId=9#"

	contentsSelector = "#data-container"
	articlesSelector = ".frame_plug"
)

var (
	bgImgRegex = regexp.MustCompile(`background-image:url\((.*)\)`)
)

// ScrapeNotices returns all notices loaded on the page into an Article slice
func ScrapeNotices() ([]models.Article, string) {
	return scrape(noticesUrl, models.NOTICE)
}

// ScrapeEvents returns all events loaded on the page into an Article slice
func ScrapeEvents() ([]models.Article, string) {
	return scrape(eventsUrl, models.EVENTS)
}

// ScrapePatchNotes returns all patch notes loaded on the page into an Article slice
func ScrapePatchNotes() ([]models.Article, string) {
	return scrape(patchNotesUrl, models.PATCHNOTES)
}

func scrape(url string, typ models.ArticleType) ([]models.Article, string) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		logrus.Fatal(err)
	}

	var articles []models.Article

	articleSelection := doc.Find(contentsSelector).Find(articlesSelector)
	cHash := getContentsHash(articleSelection.Text())

	articleSelection.Each(func(i int, s *goquery.Selection) {
		article := models.Article{Type: typ}

		articleId, exist := s.Attr("data-articleid")
		if exist {
			article.ID = convertArticleId(articleId)
		}

		feed := s.Find("a.link_feed")
		feedContents := feed.Find(".preview_text")

		articleTitle := strings.TrimSpace(feedContents.Find("strong.tit_feed").Text())
		article.Title = articleTitle

		articleDetails := strings.TrimSpace(feedContents.Find("p.txt_feed").Text())
		article.Desc = articleDetails

		imgSelector, exist := feed.Find(".preview_feed").Find("div.img").Attr("style")
		if exist {
			articleImg := bgImgRegex.FindStringSubmatch(imgSelector)[1]
			article.ImgURL = articleImg
		}

		articles = append(articles, article)
	})

	return articles, cHash
}

func convertArticleId(id string) int {
	i, err := strconv.Atoi(id)
	if err != nil {
		return -1
	}
	return i
}

func getContentsHash(contents string) string {
	h := sha1.New()
	io.WriteString(h, contents)
	return hex.EncodeToString(h.Sum(nil))
}
