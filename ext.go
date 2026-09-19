package tgbotapi

import (
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"
)

/*
*   `business_connection`
*   `business_message`
*   `edited_business_message`
*   `deleted_business_messages`
*   `guest_message`
*   `message_reaction`
*   `message_reaction_count`
*   `purchased_paid_media`
*   `chat_boost`
*   `removed_chat_boost`
*   `managed_bot`
*   `subscription`
*   `stopped_message_generation`
*   */

type BusinessBotRights struct {
	// CanReply is true if the bot can send and edit messages in the private
	// chats that had incoming messages in the last 24 hours.
	CanReply bool `json:"can_reply,omitempty"`
	// CanReadMessages is true if the bot can mark incoming private messages as read.
	CanReadMessages bool `json:"can_read_messages,omitempty"`
	// CanDeleteSentMessages is true if the bot can delete messages sent by the bot.
	CanDeleteSentMessages bool `json:"can_delete_sent_messages,omitempty"`
	// CanDeleteAllMessages is true if the bot can delete all private messages in managed chats.
	CanDeleteAllMessages bool `json:"can_delete_all_messages,omitempty"`
	// CanEditName is true if the bot can edit the first and last name of the business account.
	CanEditName bool `json:"can_edit_name,omitempty"`
	// CanEditBio is true if the bot can edit the bio of the business account.
	CanEditBio bool `json:"can_edit_bio,omitempty"`
	// CanEditProfilePhoto is true if the bot can edit the profile photo of the business account.
	CanEditProfilePhoto bool `json:"can_edit_profile_photo,omitempty"`
	// CanEditUsername is true if the bot can edit the username of the business account.
	CanEditUsername bool `json:"can_edit_username,omitempty"`
	// CanChangeGiftSettings is true if the bot can change the privacy settings pertaining to gifts for the business account.
	CanChangeGiftSettings bool `json:"can_change_gift_settings,omitempty"`
	// CanViewGiftsAndStars is true if the bot can view gifts and the amount of Telegram Stars owned by the business account.
	CanViewGiftsAndStars bool `json:"can_view_gifts_and_stars,omitempty"`
	// CanConvertGiftsToStars is true if the bot can convert regular gifts owned by the business account to Telegram Stars.
	CanConvertGiftsToStars bool `json:"can_convert_gifts_to_stars,omitempty"`
	// CanTransferAndUpgradeGifts is true if the bot can transfer and upgrade gifts owned by the business account.
	CanTransferAndUpgradeGifts bool `json:"can_transfer_and_upgrade_gifts,omitempty"`
	// CanTransferStars is true if the bot can transfer Telegram Stars received by the business account to its own account, or use them to upgrade and transfer gifts.
	CanTransferStars bool `json:"can_transfer_stars,omitempty"`
	// CanManageStories is true if the bot can post, edit and delete stories on behalf of the business account.
	CanManageStories bool `json:"can_manage_stories,omitempty"`
}

// BusinessConnection describes the connection of the bot with a business account.
type BusinessConnection struct {
	// Id is the unique identifier of the business connection.
	Id string `json:"id" gorm:"primaryKey"`
	// User is the business account user that created the business connection.
	User *User `json:"user,omitempty" gorm:"serializer:json;type:jsonb;default:null;"`
	// UserChatId is the identifier of a private chat with the user who created the business connection.
	UserChatId int64 `json:"user_chat_id"`
	// Date is the date the connection was established in Unix time.
	Date int64 `json:"date"`
	// Rights is the rights of the business bot.
	Rights *BusinessBotRights `json:"rights,omitempty" gorm:"serializer:json;type:jsonb;default:null;"`
	// IsEnabled is true if the connection is active.
	IsEnabled bool `json:"is_enabled"`
}

func (b BusinessConnection) Time() time.Time {
	if b.Date == 0 {
		return time.Time{}
	}
	return time.Unix(b.Date, 0)
}

// BusinessMessagesDeleted is received when messages are deleted from a connected business account.
type BusinessMessagesDeleted struct {
	// BusinessConnectionID is the unique identifier of the business connection.
	BusinessConnectionID string `json:"business_connection_id"`
	// Chat is information about a chat in the business account.
	Chat *Chat `json:"chat,omitempty"`
	// MessageIDs is an array of message identifiers that were deleted.
	MessageIDs []int `json:"message_ids"`
}

// MessageOrigin describes the origin of a message.
//
// It can be one of several types: user, hidden_user, chat, or channel.
type MessageOrigin struct {
	// Type is the type of the message origin: "user", "hidden_user", "chat", or "channel".
	Type string `json:"type"`
	// Date is the date the message was sent originally in Unix time.
	Date int `json:"date"`

	// --- Fields for MessageOriginUser ---
	// SenderUser is the user that sent the message originally.
	SenderUser *User `json:"sender_user,omitempty"`

	// --- Fields for MessageOriginHiddenUser ---
	// SenderUserName is the name of the user that sent the message originally.
	SenderUserName string `json:"sender_user_name,omitempty"`

	// --- Fields for MessageOriginChat ---
	// SenderChat is the chat that sent the message originally.
	SenderChat *Chat `json:"sender_chat,omitempty"`
	// AuthorSignature is the original message author signature (for anonymous admins).
	// optional -- OR CHANNEL
	AuthorSignature string `json:"author_signature,omitempty"`

	// --- Fields for MessageOriginChannel ---
	// Chat is the channel chat to which the message was originally sent.
	Chat *Chat `json:"chat,omitempty"`
	// MessageID is the unique message identifier inside the original chat.
	MessageID int `json:"message_id,omitempty"`
}

// RefundedPayment contains basic information about a refunded payment.
type RefundedPayment struct {
	// Currency three-letter ISO 4217 currency code, or "XTR" for Telegram Stars.
	Currency string `json:"currency"`
	// TotalAmount total refunded price in the smallest units of the currency.
	TotalAmount int `json:"total_amount"`
	// InvoicePayload bot specified invoice payload.
	InvoicePayload string `json:"invoice_payload"`
	// TelegramPaymentChargeID telegram payment identifier.
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	// ProviderPaymentChargeID provider payment identifier.
	// optional
	ProviderPaymentChargeID string `json:"provider_payment_charge_id,omitempty"`
}

// TextQuote contains information about the quoted part of a message that is replied to.
type TextQuote struct {
	// Text is the text of the quoted part of a message.
	Text string `json:"text"`
	// Entities are special entities that appear in the quote.
	// optional
	Entities []MessageEntity `json:"entities,omitempty"`
	// Position is the approximate quote position in the original message in UTF-16 code units.
	Position int `json:"position"`
	// IsManual is true if the quote was chosen manually by the sender.
	// optional
	IsManual bool `json:"is_manual,omitempty"`
}

// ExternalReplyInfo contains information about a message that is being replied to,
// which may come from another chat or forum topic.
type ExternalReplyInfo struct {
	// Origin is the origin of the message replied to.
	Origin MessageOrigin `json:"origin"`
	// Chat is the chat the original message belongs to (Supergroups/Channels only).
	// optional
	Chat *Chat `json:"chat,omitempty"`
	// MessageID is the unique message identifier inside the original chat.
	// optional
	MessageID *int `json:"message_id,omitempty"`
	// LinkPreviewOptions are options used for link preview generation.
	// optional
	LinkPreviewOptions json.RawMessage `json:"link_preview_options,omitempty"`
	// Animation is an animation, information about the animation.
	// optional
	Animation *Animation `json:"animation,omitempty"`
	// Audio is an audio file, information about the file.
	// optional
	Audio *Audio `json:"audio,omitempty"`
	// Document is a general file, information about the file.
	// optional
	Document *Document `json:"document,omitempty"`
	// LivePhoto is a live photo, information about the live photo.
	// optional
	LivePhoto json.RawMessage `json:"live_photo,omitempty"`
	// PaidMedia is information about paid media.
	// optional
	PaidMedia json.RawMessage `json:"paid_media,omitempty"`
	// Photo is a photo, available sizes.
	// optional
	Photo []PhotoSize `json:"photo,omitempty"`
	// Sticker is a sticker, information about the sticker.
	// optional
	Sticker *Sticker `json:"sticker,omitempty"`
	// Story is a forwarded story.
	// optional
	Story json.RawMessage `json:"story,omitempty"`
	// Video is a video, information about the video.
	// optional
	Video *Video `json:"video,omitempty"`
	// VideoNote is a video note, information about the video message.
	// optional
	VideoNote *VideoNote `json:"video_note,omitempty"`
	// Voice is a voice message, information about the file.
	// optional
	Voice *Voice `json:"voice,omitempty"`
	// HasMediaSpoiler is true if the message media is covered by a spoiler.
	// optional
	HasMediaSpoiler bool `json:"has_media_spoiler,omitempty"`
	// Checklist is a checklist.
	// optional
	Checklist json.RawMessage `json:"checklist,omitempty"`
	// Contact is a shared contact, information about the contact.
	// optional
	Contact *Contact `json:"contact,omitempty"`
	// Dice is a dice with random value.
	// optional
	Dice *Dice `json:"dice,omitempty"`
	// Game is a game, information about the game.
	// optional
	Game *Game `json:"game,omitempty"`
	// Giveaway is a scheduled giveaway.
	// optional
	Giveaway json.RawMessage `json:"giveaway,omitempty"`
	// GiveawayWinners is a giveaway with public winners.
	// optional
	GiveawayWinners json.RawMessage `json:"giveaway_winners,omitempty"`
	// Invoice is an invoice for a payment.
	// optional
	Invoice *Invoice `json:"invoice,omitempty"`
	// Location is a shared location.
	// optional
	Location *Location `json:"location,omitempty"`
	// Poll is a native poll.
	// optional
	Poll *Poll `json:"poll,omitempty"`
	// Venue is a venue.
	// optional
	Venue *Venue `json:"venue,omitempty"`
}

// ReplyParameters describes reply parameters for the message that is being sent.
type ReplyParameters struct {
	// MessageID is the identifier of the message to be replied to.
	// Required if ephemeral_message_id isn't specified.
	// optional
	MessageID *int `json:"message_id,omitempty"`
	// ChatID is the identifier for the chat or username of the bot/supergroup/channel.
	// optional
	ChatID any `json:"chat_id,omitempty"` // Can be int64 or string
	// EphemeralMessageID is the identifier of the incoming ephemeral message.
	// Required if message_id isn't specified.
	// optional
	EphemeralMessageID *int `json:"ephemeral_message_id,omitempty"`
	// AllowSendingWithoutReply is true if the message should be sent even if the reply target isn't found.
	// optional
	AllowSendingWithoutReply bool `json:"allow_sending_without_reply,omitempty"`
	// Quote is the quoted part of the message to be replied to (0-1024 chars).
	// optional
	Quote string `json:"quote,omitempty"`
	// QuoteParseMode is the mode for parsing entities in the quote.
	// optional
	QuoteParseMode string `json:"quote_parse_mode,omitempty"`
	// QuoteEntities is a JSON-serialized list of special entities appearing in the quote.
	// optional
	QuoteEntities []MessageEntity `json:"quote_entities,omitempty"`
	// QuotePosition is the position of the quote in the original message (UTF-16).
	// optional
	QuotePosition *int `json:"quote_position,omitempty"`
	// ChecklistTaskID is the identifier of the specific checklist task being replied to.
	// optional
	ChecklistTaskID *int `json:"checklist_task_id,omitempty"`
	// PollOptionID is the persistent identifier of the specific poll option being replied to.
	// optional
	PollOptionID string `json:"poll_option_id,omitempty"`
}

// DirectMessagesTopic describes a topic of a direct messages chat.
type DirectMessagesTopic struct {
	// TopicID is the unique identifier of the topic.
	// We use int64 because the spec states it can have up to 52 significant bits,
	// which fits safely within a signed 64-bit integer.
	TopicID int64 `json:"topic_id"`

	// User is the information about the user that created the topic.
	// We use a pointer (*User) to represent the "Optional" nature
	// of this field in the specification.
	User *User `json:"user,omitempty"`
}

// Invoice contains the parameters for sending or creating an invoice link.
type Invoice struct {
	// ChatID is the unique identifier for the target chat or username of the target bot,
	// supergroup or channel in the format @username.
	// Can be int64 or string.
	ChatID interface{} `json:"chat_id"`

	// BusinessConnectionID is the unique identifier of the business connection on behalf
	// of which the link will be created. For payments in Telegram Stars only.
	// optional
	BusinessConnectionID string `json:"business_connection_id,omitempty"`

	// MessageThreadID is the unique identifier for the target message thread (topic).
	// optional
	MessageThreadID *int `json:"message_thread_id,omitempty"`

	// DirectMessagesTopicID is the identifier of the direct messages topic.
	// required if the message is sent to a direct messages chat.
	// optional
	DirectMessagesTopicID *int `json:"direct_messages_topic_id,omitempty"`

	// Title is the product name (1-32 characters).
	Title string `json:"title"`
	// Description is the product description (1-255 characters).
	Description string `json:"description"`
	// Payload is a bot-defined invoice payload (1-128 bytes).
	Payload string `json:"payload"`

	// ProviderToken is the payment provider token, obtained via @BotFather.
	// Pass an empty string for payments in Telegram Stars.
	// optional
	ProviderToken string `json:"provider_token,omitempty"`
	// Currency is a three-letter ISO 4217 currency code. Pass “XTR” for Telegram Stars.
	Currency string `json:"currency"`
	// Prices is a list of labeled prices (must contain exactly one item for Telegram Stars).
	Prices []LabeledPrice `json:"prices"`

	// MaxTipAmount is the maximum accepted amount for tips in the smallest units.
	// Defaults to 0. Not supported for payments in Telegram Stars.
	// optional
	MaxTipAmount int `json:"max_tip_amount,omitempty"`
	// SuggestedTipAmounts is an array of suggested tip amounts in the smallest units.
	// optional
	SuggestedTipAmounts []int `json:"suggested_tip_amounts,omitempty"`

	// StartParameter is a unique deep-linking parameter.
	// optional
	StartParameter string `json:"start_parameter,omitempty"`

	// ProviderData is JSON-serialized data shared with the payment provider.
	// optional
	ProviderData string `json:"provider_data,omitempty"`

	// PhotoURL is the URL of the product photo.
	// optional
	PhotoURL string `json:"photo_url,omitempty"`
	// PhotoSize is the photo size in bytes.
	// optional
	PhotoSize *int `json:"photo_size,omitempty"`
	// PhotoWidth is the photo width.
	// optional
	PhotoWidth *int `json:"photo_width,omitempty"`
	// PhotoHeight is the photo height.
	// optional
	PhotoHeight *int `json:"photo_height,omitempty"`

	// NeedName requires the user's full name. Ignored for Telegram Stars.
	// optional
	NeedName bool `json:"need_name,omitempty"`
	// NeedPhoneNumber requires the user's phone number. Ignored for Telegram Stars.
	// optional
	NeedPhoneNumber bool `json:"need_phone_number,omitempty"`
	// NeedEmail requires the user's email address. Ignored for Telegram Stars.
	// optional
	NeedEmail bool `json:"need_email,omitempty"`
	// NeedShippingAddress requires the user's shipping address. Ignored for Telegram Stars.
	// optional
	NeedShippingAddress bool `json:"need_shipping_address,omitempty"`
	// SendPhoneNumberToProvider sends the phone number to the provider. Ignored for Telegram Stars.
	// optional
	SendPhoneNumberToProvider bool `json:"send_phone_number_to_provider,omitempty"`
	// SendEmailToProvider sends the email address to the provider. Ignored for Telegram Stars.
	// optional
	SendEmailToProvider bool `json:"send_email_to_provider,omitempty"`
	// IsFlexible is true if the final price depends on the shipping method. Ignored for Telegram Stars.
	// optional
	IsFlexible bool `json:"is_flexible,omitempty"`

	// DisableNotification sends the message silently.
	// optional
	DisableNotification bool `json:"disable_notification,omitempty"`
	// ProtectContent protects the message from forwarding and saving.
	// optional
	ProtectContent bool `json:"protect_content,omitempty"`
	// AllowPaidBroadcast allows up to 1000 messages per second (requires Stars).
	// optional
	AllowPaidBroadcast bool `json:"allow_paid_broadcast,omitempty"`

	// MessageEffectID is a unique identifier for the message effect.
	// optional
	MessageEffectID string `json:"message_effect_id,omitempty"`

	// SuggestedPostParameters contains parameters for the suggested post.
	// optional
	SuggestedPostParameters *SuggestedPostParameters `json:"suggested_post_parameters,omitempty"`
	// ReplyParameters describes the message to be replied to.
	// optional
	ReplyParameters *ReplyParameters `json:"reply_parameters,omitempty"`
}
type SuggestedPostParameters struct {
	Price    *SuggestedPostPrice `json:"price,omitempty"`
	SendDate int64               `json:"send_date,omitempty"`
}
type SuggestedPostPrice struct {
	Currency string `json:"currency"`
	Ammount  int64  `json:"amount"`
}

// OldInvoice contains basic information about an invoice.
type OldInvoice struct {
	// Title product name
	Title string `json:"title"`
	// Description product description
	Description string `json:"description"`
	// StartParameter unique bot deep-linking parameter that can be used to generate this invoice
	StartParameter string `json:"start_parameter"`
	// Currency three-letter ISO 4217 currency code
	// (see https://core.telegram.org/bots/payments#supported-currencies)
	Currency string `json:"currency"`
	// TotalAmount total price in the smallest units of the currency (integer, not float/double).
	// For example, for a price of US$ 1.45 pass amount = 145.
	// See the exp parameter in currencies.json
	// (https://core.telegram.org/bots/payments/currencies.json),
	// it shows the number of digits past the decimal point
	// for each currency (2 for the majority of currencies).
	TotalAmount int `json:"total_amount"`
}

// InputInvoiceMessageContent represents the content of an invoice message to be
// sent as the result of an inline query.
type InputInvoiceMessageContent struct {
	// Product name, 1-32 characters
	Title string `json:"title"`
	// Product description, 1-255 characters
	Description string `json:"description"`
	// Bot-defined invoice payload, 1-128 bytes. This will not be displayed to
	// the user, use for your internal processes.
	Payload string `json:"payload"`
	// Payment provider token, obtained via Botfather
	ProviderToken string `json:"provider_token"`
	// Three-letter ISO 4217 currency code
	Currency string `json:"currency"`
	// Price breakdown, a JSON-serialized list of components (e.g. product
	// price, tax, discount, delivery cost, delivery tax, bonus, etc.)
	Prices []LabeledPrice `json:"prices"`
	// The maximum accepted amount for tips in the smallest units of the
	// currency (integer, not float/double).
	//
	// optional
	MaxTipAmount int `json:"max_tip_amount,omitempty"`
	// An array of suggested amounts of tip in the smallest units of the
	// currency (integer, not float/double). At most 4 suggested tip amounts can
	// be specified. The suggested tip amounts must be positive, passed in a
	// strictly increased order and must not exceed max_tip_amount.
	//
	// optional
	SuggestedTipAmounts []int `json:"suggested_tip_amounts,omitempty"`
	// A JSON-serialized object for data about the invoice, which will be shared
	// with the payment provider. A detailed description of the required fields
	// should be provided by the payment provider.
	//
	// optional
	ProviderData string `json:"provider_data,omitempty"`
	// URL of the product photo for the invoice. Can be a photo of the goods or
	// a marketing image for a service. People like it better when they see what
	// they are paying for.
	//
	// optional
	PhotoURL string `json:"photo_url,omitempty"`
	// Photo size
	//
	// optional
	PhotoSize int `json:"photo_size,omitempty"`
	// Photo width
	//
	// optional
	PhotoWidth int `json:"photo_width,omitempty"`
	// Photo height
	//
	// optional
	PhotoHeight int `json:"photo_height,omitempty"`
	// Pass True, if you require the user's full name to complete the order
	//
	// optional
	NeedName bool `json:"need_name,omitempty"`
	// Pass True, if you require the user's phone number to complete the order
	//
	// optional
	NeedPhoneNumber bool `json:"need_phone_number,omitempty"`
	// Pass True, if you require the user's email address to complete the order
	//
	// optional
	NeedEmail bool `json:"need_email,omitempty"`
	// Pass True, if you require the user's shipping address to complete the order
	//
	// optional
	NeedShippingAddress bool `json:"need_shipping_address,omitempty"`
	// Pass True, if user's phone number should be sent to provider
	//
	// optional
	SendPhoneNumberToProvider bool `json:"send_phone_number_to_provider,omitempty"`
	// Pass True, if user's email address should be sent to provider
	//
	// optional
	SendEmailToProvider bool `json:"send_email_to_provider,omitempty"`
	// Pass True, if the final price depends on the shipping method
	//
	// optional
	IsFlexible bool `json:"is_flexible,omitempty"`
}

type ManagedBotConfig struct {
	UserID int64 `json:"user_id"`
}

func (b *BotAPI) GetManagedBotToken(otherBotUserID int64) (string, error) {
	got, err := b.MakeRequestJSON("getManagedBotToken", map[string]any{"user_id": otherBotUserID})
	token := ""
	err = json.Unmarshal(got.Result, &token)
	if err != nil {
		return "", Errorf("%w", err)
	}
	return token, nil
}

func (b *BotAPI) ReplaceManagedBotToken(otherBotUserID int64) (string, error) {
	got, err := b.MakeRequestJSON("replaceManagedBotToken", map[string]any{"user_id": otherBotUserID})
	token := ""
	err = json.Unmarshal(got.Result, &token)
	if err != nil {
		return "", Errorf("%w", err)
	}
	return token, nil
}

type BotAccessSettings struct {
	IsAccessRestricted bool   `json:"is_access_restricted"` // True, if only selected users can access the bot. The bot's owner can always access it.
	AddedUsers         []User // Optional. The list of other users who have access to the bot if the access is restricted
}

func (b *BotAPI) GetManagedBotSettings(otherBotUserID int64) (BotAccessSettings, error) {
	got, err := b.MakeRequestJSON("replaceManagedBotToken", map[string]any{"user_id": otherBotUserID})
	var access BotAccessSettings
	err = json.Unmarshal(got.Result, &access)
	if err != nil {
		return BotAccessSettings{}, Errorf("%w", err)
	}
	return access, nil
}

func (bot *BotAPI) GetBusinessConnection(businessConnectionID string) (BusinessConnection, error) {
	got, err := bot.MakeRequestJSON("getBusinessConnection", map[string]string{"business_connection_id": businessConnectionID})
	if err != nil {
		return BusinessConnection{}, Errorf("%w", err)
	}
	conn := BusinessConnection{}
	err = json.Unmarshal(got.Result, &conn)
	if err != nil {
		return BusinessConnection{}, Errorf("%w", err)
	}
	return conn, nil
}

// 429 rate limit
var retryRegex = regexp.MustCompile(`retry after\s+(\d+)`)

// ParseRetryAfter does not sleep
func ParseRetryAfter(err error) int {
	if err == nil {
		return 0
	}
	errStr := err.Error()
	matches := retryRegex.FindStringSubmatch(errStr)
	if len(matches) != 2 {
		return 0
	}
	seconds, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return seconds
}

var gratelimit atomic.Pointer[time.Time]

func BumpRateLimit(err error) {
	if err == nil {
		return
	}
	nsec := time.Duration(ParseRetryAfter(err))
	gratelimit.Store(new(time.Now().Add(time.Second * nsec)))
}

// maybewait sleeps
func maybewait() {
	until := gratelimit.Load()
	if until == nil {
		return
	}
	if (*until).IsZero() {
		return
	}
	d := time.Until(*until)
	slog.Info("SLEEP RATE LIMIT", "d", d.String())
	time.Sleep(d)
}
