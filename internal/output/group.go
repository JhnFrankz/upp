package output

// Group is a section of the list/selector output: a manager header line
// (empty for standalone tools) and the list rows under it.
//
// For a manager group, Header is the manager's display name and Items holds
// the manager's own row first followed by the tools it owns on the current
// platform. For the trailing standalone group, Header is empty and Items
// holds the standalone tool rows.
type Group struct {
	Header string
	Items  []ListEntry
}
