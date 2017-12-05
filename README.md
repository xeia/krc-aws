# Kings-Raid-Crawler
> Late in getting notices and lazy in browsing to the cafe?<br>
> Or tired of refreshing the cafe every Tuesday for the maintenance notice or every Thursday for the patch notes?<br>
> Perhaps its better to save time for raiding and let the crawling do the job!

## Features
- Crawls for articles on the cafe every hour via a cron job
- Store articles into AWS DynamoDB
- Serverless implementation
- When a new article is published, sends a Telegram message to all users with the link 
- Telegram Bot commands to request for article in different categories

## Todo
- [ ] Setup storage models
- [ ] Setup webpage crawl and parsing
- [ ] Setup cron worker
- [ ] Setup AWS credentials
- [ ] Setup serverless implementation
- [ ] Get Telegram Bot credentials
- [ ] Deployment

## Setup
### Requirements
- Golang 1.9
- AWS Lambda, CloudWatch, DynamoDB
- Telegram Bot
- [Up](https://up.docs.apex.sh)

### Process
> Build 

## Future
- [ ] To add Discord functionalities
- [ ] Add hero/uw details
- [ ] Add artifact details
