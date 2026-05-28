package notify

import "context"

// Notifier delivers alert reports to a messaging channel.
type Notifier interface {
	SendMessage(ctx context.Context, chatID string, text string) error
	SendPhoto(ctx context.Context, chatID string, photo []byte, caption string) error
	SendDocument(ctx context.Context, chatID string, doc []byte, filename string, caption string) error
}
