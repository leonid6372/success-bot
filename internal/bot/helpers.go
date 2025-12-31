package bot

import "github.com/leonid6372/success-bot/internal/common/domain"

func (b *Bot) closeInstrument(user *domain.User) {
	close(*user.Metadata.InstrumentDone)
	user.Metadata.InstrumentDone = nil
}
