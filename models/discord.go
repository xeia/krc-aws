package models

// DiscordFooter ...
type DiscordFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url"`
}

// DiscordImage ...
type DiscordImage struct {
	URL string `json:"url"`
}

// DiscordThumbnail ...
type DiscordThumbnail struct {
	URL string `json:"url"`
}

// DiscordAuthor ...
type DiscordAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	IconURL string `json:"icon_url"`
}

// DiscordField ...
type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// DiscordEmbed ...
type DiscordEmbed struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	URL         string           `json:"url"`
	Timestamp   string           `json:"timestamp"`
	Color       int              `json:"color"`
	Footer      DiscordFooter    `json:"footer"`
	Image       DiscordImage     `json:"image"`
	Thumbnail   DiscordThumbnail `json:"thumbnail"`
	Author      DiscordAuthor    `json:"author"`
	Fields      []DiscordField   `json:"fields"`
}

// DiscordHookMessage
type DiscordHookMessage struct {
	Content string         `json:"content"`
	Embeds  []DiscordEmbed `json:"embeds"`
}
