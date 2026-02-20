package resolve

// ChannelResolver provides channel name/ID resolution. Tool handlers and
// helpers accept this interface rather than the concrete *Resolver type.
type ChannelResolver interface {
	ChannelName(id string) string
	ChannelID(name string) (string, error)
}

// Compile-time assertion: *Resolver satisfies ChannelResolver.
var _ ChannelResolver = (*Resolver)(nil)
