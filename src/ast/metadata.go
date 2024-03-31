package ast

import "fmt"

type MetadataAttachment interface {
	String() string
	// kind of the attachement (e.g. "range", "constant_info", "type_info", etc.)
	// should be a unique string for each kind of metadata as it is the key in the
	// attachments map
	MetaKind() string
}

type Metadata struct {
	// Attachements sorted by kind
	Attachments map[string]MetadataAttachment
}

// TODO: make this good
func (md *Metadata) String() string {
	return fmt.Sprintf("Metadata{ %v }", md.Attachments)
}
