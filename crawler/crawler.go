package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"fmt"
	"strings"
	"regexp"
)

const (
	cafeBase = "https://www.plug.game/kingsraid-en"
	noticesUrl = cafeBase + "/posts?menuId=1#"
	eventsUrl = cafeBase + "/posts?menuId=2#"
	patchNotesUrl = cafeBase + "/posts?menuId=9#"

	contentsSelector = "#data-container"
	articlesSelector = ".frame_plug"
)

var (
	bgImgRegex = regexp.MustCompile(`background-image:url\((.*)\)`)
)

// ScrapeNotices returns all notices loaded on the page into an Article slice
func ScrapeNotices() {
	scrape(noticesUrl)
}

// ScrapeEvents returns all events loaded on the page into an Article slice
func ScrapeEvents() {
	scrape(eventsUrl)
}

// ScrapePatchNotes returns all patch notes loaded on the page into an Article slice
func ScrapePatchNotes() {
	scrape(patchNotesUrl)
}

func scrape(url string) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		logrus.Fatal(err)
	}
	doc.Find(contentsSelector).Find(articlesSelector).Each(func(i int, s *goquery.Selection) {
		articleId, exist := s.Attr("data-articleid")
		if exist {
			fmt.Println(articleId)
		}
		article := s.Find("a.link_feed")
		articleContents := article.Find(".preview_text")
		articleTitle := strings.TrimSpace(articleContents.Find("strong.tit_feed").Text())
		articleDetails := strings.TrimSpace(articleContents.Find("p.txt_feed").Text())
		fmt.Println(articleTitle)
		fmt.Println(articleDetails)
		imgSelector, exist := article.Find(".preview_feed").Find("div.img").Attr("style")
		if exist {
			articleImg := bgImgRegex.FindStringSubmatch(imgSelector)[1]
			fmt.Println(articleImg)
		}
	})
}

func main() {
	ScrapePatchNotes()
}