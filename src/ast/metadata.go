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

var _ *MetadataAnnotated = nil

// TODO: make this good
func (md *Metadata) String() string {
	return fmt.Sprintf("Metadata{ %v }", md.Attachments)
}

// Annotator is a Visitor that can be used to annotate an AST with Metadata
type Annotator Visitor

type MetadataAnnotated interface {
	GetMetadata() Metadata
	GetMetadataByKind(MetadataKind) (MetadataAttachment, bool)
	HasMetadata(MetadataKind) bool
	SetMetadataAttachement(MetadataAttachment)
	RemoveMetadataAttachment(MetadataKind)
}

func (md *Metadata) GetMetadata() Metadata {
	return *md
}

func (md *Metadata) GetMetadataByKind(kind MetadataKind) (MetadataAttachment, bool) {
	if md == nil {
		return nil, false
	}
	att, ok := md.Attachments[kind]
	return att, ok
}

func (md *Metadata) HasMetadata(kind MetadataKind) bool {
	_, ok := md.GetMetadataByKind(kind)
	return ok
}

func (md *Metadata) SetMetadataAttachement(attachment MetadataAttachment) {
	if md == nil {
		return
	}

	if md.Attachments == nil {
		md.Attachments = make(map[MetadataKind]MetadataAttachment)
	}
	md.Attachments[attachment.Kind()] = attachment
}

func (md *Metadata) RemoveMetadataAttachment(kind MetadataKind) {
	if md == nil {
		return
	}

	delete(md.Attachments, kind)
}
