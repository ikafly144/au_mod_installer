package discord

var (
	ErrNotLoggedIn = &DiscordError{"not logged in"}
)

type DiscordError struct {
	Message string
}

func (e *DiscordError) Error() string {
	return e.Message
}
