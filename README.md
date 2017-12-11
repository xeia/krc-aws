# Kings-Raid-Crawler
> Late in getting notices and lazy in browsing to the cafe?<br>
> Or tired of refreshing the cafe every Tuesday for the maintenance notice or every Thursday for the patch notes?<br>
> Perhaps its better to save time for raiding and let the crawling do the job!

## Features
- Crawls for articles on the cafe every hour via a cron job
- Store articles into AWS DynamoDB
- Serverless implementation
- When a new article is published, sends message to web hooks

## Setup
### Requirements
- Golang 1.9
- AWS Lambda, CloudWatch, DynamoDB
- [Sparta](http://gosparta.io)
- [goquery](https://godoc.org/github.com/PuerkitoBio/goquery)
- Discord Webhook

### Process
0. Setup requirements and provision the necessary roles/policies on AWS
1. Clone this repo
2. Create a `.env` file with the following fields:
```
DISCORD_WEBHOOK=<WEBHOOK_URL>
DYNAMODB_DBSTREAM=<DYNAMODB_STREAM_ARN>
```

3. Enable/Disable Discord/Telegram functionalities
4. Modify the IAM definitions for the functions to those that you have provisioned
5. Setup a S3 Bucket for code storage and store as $S3_BUCKET
6. Run the provision command:
> go run *.go provision --level info --s3Bucket $S3_BUCKET
7. Go onto AWS and view the consoles for the relevant functions, making changes as necessary
8. *(Optional)* Setup CloudWatch Alarm with a schedule to invoke ScrapeAll

