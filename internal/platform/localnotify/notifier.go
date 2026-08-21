package localnotify

import (
	"context"
	"sync"

	"github.com/wyw14/cry-055/internal/domain"
)

type Notifier struct {
	mu   sync.Mutex
	sent []domain.Alert
}

func New() *Notifier { return &Notifier{} }
func (n *Notifier) Notify(ctx context.Context, alert domain.Alert) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, alert)
	return nil
}
func (n *Notifier) Sent() []domain.Alert {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]domain.Alert(nil), n.sent...)
}
func (n *Notifier) Reset() { n.mu.Lock(); defer n.mu.Unlock(); n.sent = nil }
