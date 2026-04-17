package gatekeeper

// LineWebhookRequest represents LINE webhook request body.
type LineWebhookRequest struct {
	Events []LineEvent `json:"events"`
}

// LineEvent represents a single event in LINE webhook.
type LineEvent struct {
	Type       string      `json:"type"`
	ReplyToken string      `json:"replyToken"`
	Message    LineMessage `json:"message"`
	Source     LineSource  `json:"source"`
}

// LineMessage represents a message in LINE event.
type LineMessage struct {
	Type    string       `json:"type"`
	Text    string       `json:"text"`
	Mention *LineMention `json:"mention,omitempty"`
}

// LineMention represents mention information in LINE message.
type LineMention struct {
	Mentionees []LineMentionee `json:"mentionees"`
}

// LineMentionee represents a single mentioned user.
type LineMentionee struct {
	Index  int    `json:"index"`
	Length int    `json:"length"`
	UserID string `json:"userId"`
}

// LineSource represents the source of LINE event.
type LineSource struct {
	Type    string `json:"type"` // "user", "group", "room"
	UserID  string `json:"userId"`
	GroupID string `json:"groupId,omitempty"`
	RoomID  string `json:"roomId,omitempty"`
}
