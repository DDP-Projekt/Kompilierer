package ast

import "fmt"

type MetadataKind string

type MetadataAttachment interface {
	String() string
	// kind of the attachement (e.g. "range", "constant_info", "type_info", etc. or any custom string)
	// should be a unique string for each kind of metadata as it is the key in the
	// attachments map
	Kind() MetadataKind
}

// a collection of MetadataAttachments for a node
type Metadata struct {
	// Attachements stored by kind
	Attachments map[MetadataKind]MetadataAttachment
}

// TODO: make this good
func (md *Metadata) String() string {
	return fmt.Sprintf("Metadata{ %v }", md.Attachments)
}

// Annotator is a Visitor that can be used to annotate an AST with Metadata
type Annotator Visitor
