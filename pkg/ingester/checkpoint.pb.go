// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pkg/ingester/checkpoint.proto

package ingester

import (
	bytes "bytes"
	fmt "fmt"
	_ "github.com/cortexproject/cortex/pkg/cortexpb"
	github_com_cortexproject_cortex_pkg_cortexpb "github.com/cortexproject/cortex/pkg/cortexpb"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Chunk is a {de,}serializable intermediate type for chunkDesc which allows
// efficient loading/unloading to disk during WAL checkpoint recovery.
type Chunk struct {
	From        time.Time `protobuf:"bytes,1,opt,name=from,proto3,stdtime" json:"from"`
	To          time.Time `protobuf:"bytes,2,opt,name=to,proto3,stdtime" json:"to"`
	FlushedAt   time.Time `protobuf:"bytes,3,opt,name=flushedAt,proto3,stdtime" json:"flushedAt"`
	LastUpdated time.Time `protobuf:"bytes,4,opt,name=lastUpdated,proto3,stdtime" json:"lastUpdated"`
	Closed      bool      `protobuf:"varint,5,opt,name=closed,proto3" json:"closed,omitempty"`
	Synced      bool      `protobuf:"varint,6,opt,name=synced,proto3" json:"synced,omitempty"`
	// data to be unmarshaled into a MemChunk
	Data []byte `protobuf:"bytes,7,opt,name=data,proto3" json:"data,omitempty"`
	// data to be unmarshaled into a MemChunk's headBlock
	Head []byte `protobuf:"bytes,8,opt,name=head,proto3" json:"head,omitempty"`
}

func (m *Chunk) Reset()      { *m = Chunk{} }
func (*Chunk) ProtoMessage() {}
func (*Chunk) Descriptor() ([]byte, []int) {
	return fileDescriptor_00f4b7152db9bdb5, []int{0}
}
func (m *Chunk) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Chunk) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Chunk.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Chunk) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Chunk.Merge(m, src)
}
func (m *Chunk) XXX_Size() int {
	return m.Size()
}
func (m *Chunk) XXX_DiscardUnknown() {
	xxx_messageInfo_Chunk.DiscardUnknown(m)
}

var xxx_messageInfo_Chunk proto.InternalMessageInfo

func (m *Chunk) GetFrom() time.Time {
	if m != nil {
		return m.From
	}
	return time.Time{}
}

func (m *Chunk) GetTo() time.Time {
	if m != nil {
		return m.To
	}
	return time.Time{}
}

func (m *Chunk) GetFlushedAt() time.Time {
	if m != nil {
		return m.FlushedAt
	}
	return time.Time{}
}

func (m *Chunk) GetLastUpdated() time.Time {
	if m != nil {
		return m.LastUpdated
	}
	return time.Time{}
}

func (m *Chunk) GetClosed() bool {
	if m != nil {
		return m.Closed
	}
	return false
}

func (m *Chunk) GetSynced() bool {
	if m != nil {
		return m.Synced
	}
	return false
}

func (m *Chunk) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Chunk) GetHead() []byte {
	if m != nil {
		return m.Head
	}
	return nil
}

// Series is a {de,}serializable intermediate type for Series.
type Series struct {
	UserID string `protobuf:"bytes,1,opt,name=userID,proto3" json:"userID,omitempty"`
	// post mapped fingerprint is necessary because subsequent wal writes will reference it.
	Fingerprint uint64                                                      `protobuf:"varint,2,opt,name=fingerprint,proto3" json:"fingerprint,omitempty"`
	Labels      []github_com_cortexproject_cortex_pkg_cortexpb.LabelAdapter `protobuf:"bytes,3,rep,name=labels,proto3,customtype=github.com/cortexproject/cortex/pkg/cortexpb.LabelAdapter" json:"labels"`
	Chunks      []Chunk                                                     `protobuf:"bytes,4,rep,name=chunks,proto3" json:"chunks"`
	// Last timestamp of the last chunk.
	To       time.Time `protobuf:"bytes,5,opt,name=to,proto3,stdtime" json:"to"`
	LastLine string    `protobuf:"bytes,6,opt,name=lastLine,proto3" json:"lastLine,omitempty"`
}

func (m *Series) Reset()      { *m = Series{} }
func (*Series) ProtoMessage() {}
func (*Series) Descriptor() ([]byte, []int) {
	return fileDescriptor_00f4b7152db9bdb5, []int{1}
}
func (m *Series) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Series) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Series.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Series) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Series.Merge(m, src)
}
func (m *Series) XXX_Size() int {
	return m.Size()
}
func (m *Series) XXX_DiscardUnknown() {
	xxx_messageInfo_Series.DiscardUnknown(m)
}

var xxx_messageInfo_Series proto.InternalMessageInfo

func (m *Series) GetUserID() string {
	if m != nil {
		return m.UserID
	}
	return ""
}

func (m *Series) GetFingerprint() uint64 {
	if m != nil {
		return m.Fingerprint
	}
	return 0
}

func (m *Series) GetChunks() []Chunk {
	if m != nil {
		return m.Chunks
	}
	return nil
}

func (m *Series) GetTo() time.Time {
	if m != nil {
		return m.To
	}
	return time.Time{}
}

func (m *Series) GetLastLine() string {
	if m != nil {
		return m.LastLine
	}
	return ""
}

func init() {
	proto.RegisterType((*Chunk)(nil), "loki_ingester.Chunk")
	proto.RegisterType((*Series)(nil), "loki_ingester.Series")
}

func init() { proto.RegisterFile("pkg/ingester/checkpoint.proto", fileDescriptor_00f4b7152db9bdb5) }

var fileDescriptor_00f4b7152db9bdb5 = []byte{
	// 492 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x52, 0xbd, 0x8e, 0xd3, 0x4c,
	0x14, 0xf5, 0x24, 0x8e, 0x3f, 0x67, 0xf2, 0xd1, 0x0c, 0x08, 0x8d, 0x22, 0x31, 0xb1, 0xb6, 0x4a,
	0x83, 0x2d, 0x05, 0x0a, 0x68, 0x90, 0x62, 0x10, 0x12, 0xd2, 0x16, 0xc8, 0x40, 0x43, 0x83, 0xfc,
	0x33, 0xb1, 0x4d, 0x1c, 0x8f, 0x35, 0x33, 0x96, 0xa0, 0xe3, 0x11, 0xf6, 0x31, 0x78, 0x04, 0x1e,
	0x61, 0xcb, 0x94, 0x2b, 0x90, 0x16, 0xe2, 0x34, 0x94, 0xfb, 0x08, 0x68, 0xc6, 0x36, 0x1b, 0x4a,
	0x77, 0xf7, 0x9c, 0x7b, 0x8f, 0xcf, 0xf5, 0x9d, 0x03, 0x1f, 0x54, 0xdb, 0xd4, 0xcb, 0xcb, 0x94,
	0x0a, 0x49, 0xb9, 0x17, 0x67, 0x34, 0xde, 0x56, 0x2c, 0x2f, 0xa5, 0x5b, 0x71, 0x26, 0x19, 0xba,
	0x53, 0xb0, 0x6d, 0xfe, 0xa1, 0xef, 0xcf, 0x17, 0x29, 0x63, 0x69, 0x41, 0x3d, 0xdd, 0x8c, 0xea,
	0x8d, 0x27, 0xf3, 0x1d, 0x15, 0x32, 0xdc, 0x55, 0xed, 0xfc, 0xfc, 0x61, 0x9a, 0xcb, 0xac, 0x8e,
	0xdc, 0x98, 0xed, 0xbc, 0x94, 0xa5, 0xec, 0x76, 0x52, 0x21, 0x0d, 0x74, 0xd5, 0x8d, 0x3f, 0x3d,
	0x19, 0x8f, 0x19, 0x97, 0xf4, 0x53, 0xc5, 0xd9, 0x47, 0x1a, 0xcb, 0x0e, 0x79, 0x6a, 0xbb, 0xae,
	0x11, 0x75, 0x45, 0x2b, 0x3d, 0xfb, 0x31, 0x82, 0x93, 0xe7, 0x59, 0x5d, 0x6e, 0xd1, 0x13, 0x68,
	0x6e, 0x38, 0xdb, 0x61, 0xe0, 0x80, 0xe5, 0x6c, 0x35, 0x77, 0xdb, 0x1d, 0xdd, 0xde, 0xd9, 0x7d,
	0xdb, 0xef, 0xe8, 0xdb, 0x97, 0xd7, 0x0b, 0xe3, 0xe2, 0xe7, 0x02, 0x04, 0x5a, 0x81, 0x1e, 0xc3,
	0x91, 0x64, 0x78, 0x34, 0x40, 0x37, 0x92, 0x0c, 0xf9, 0x70, 0xba, 0x29, 0x6a, 0x91, 0xd1, 0x64,
	0x2d, 0xf1, 0x78, 0x80, 0xf8, 0x56, 0x86, 0x5e, 0xc2, 0x59, 0x11, 0x0a, 0xf9, 0xae, 0x4a, 0x42,
	0x49, 0x13, 0x6c, 0x0e, 0xf8, 0xca, 0xa9, 0x10, 0xdd, 0x87, 0x56, 0x5c, 0x30, 0x41, 0x13, 0x3c,
	0x71, 0xc0, 0xd2, 0x0e, 0x3a, 0xa4, 0x78, 0xf1, 0xb9, 0x8c, 0x69, 0x82, 0xad, 0x96, 0x6f, 0x11,
	0x42, 0xd0, 0x4c, 0x42, 0x19, 0xe2, 0xff, 0x1c, 0xb0, 0xfc, 0x3f, 0xd0, 0xb5, 0xe2, 0x32, 0x1a,
	0x26, 0xd8, 0x6e, 0x39, 0x55, 0x9f, 0x7d, 0x1b, 0x41, 0xeb, 0x0d, 0xe5, 0x39, 0x15, 0xea, 0x53,
	0xb5, 0xa0, 0xfc, 0xd5, 0x0b, 0x7d, 0xe0, 0x69, 0xd0, 0x21, 0xe4, 0xc0, 0xd9, 0x46, 0x05, 0x83,
	0x57, 0x3c, 0x2f, 0xa5, 0xbe, 0xa2, 0x19, 0x9c, 0x52, 0xa8, 0x84, 0x56, 0x11, 0x46, 0xb4, 0x10,
	0x78, 0xec, 0x8c, 0x97, 0xb3, 0xd5, 0x5d, 0xb7, 0x7f, 0x4a, 0xf7, 0x5c, 0xf1, 0xaf, 0xc3, 0x9c,
	0xfb, 0x6b, 0xf5, 0x63, 0xdf, 0xaf, 0x17, 0x83, 0xa2, 0xd0, 0xea, 0xd7, 0x49, 0x58, 0x49, 0xca,
	0x83, 0xce, 0x05, 0xad, 0xa0, 0x15, 0xab, 0x44, 0x08, 0x6c, 0x6a, 0xbf, 0x7b, 0xee, 0x3f, 0xe9,
	0x75, 0x75, 0x5c, 0x7c, 0x53, 0x19, 0x06, 0xdd, 0x64, 0x17, 0x81, 0xc9, 0xc0, 0x08, 0xcc, 0xa1,
	0xad, 0x5e, 0xe1, 0x3c, 0x2f, 0xa9, 0x3e, 0xf0, 0x34, 0xf8, 0x8b, 0xfd, 0x67, 0xfb, 0x03, 0x31,
	0xae, 0x0e, 0xc4, 0xb8, 0x39, 0x10, 0xf0, 0xa5, 0x21, 0xe0, 0x6b, 0x43, 0xc0, 0x65, 0x43, 0xc0,
	0xbe, 0x21, 0xe0, 0x57, 0x43, 0xc0, 0xef, 0x86, 0x18, 0x37, 0x0d, 0x01, 0x17, 0x47, 0x62, 0xec,
	0x8f, 0xc4, 0xb8, 0x3a, 0x12, 0xe3, 0xbd, 0xdd, 0x6f, 0x19, 0x59, 0xda, 0xfd, 0xd1, 0x9f, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xae, 0x13, 0x93, 0xc4, 0x9a, 0x03, 0x00, 0x00,
}

func (this *Chunk) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Chunk)
	if !ok {
		that2, ok := that.(Chunk)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if !this.From.Equal(that1.From) {
		return false
	}
	if !this.To.Equal(that1.To) {
		return false
	}
	if !this.FlushedAt.Equal(that1.FlushedAt) {
		return false
	}
	if !this.LastUpdated.Equal(that1.LastUpdated) {
		return false
	}
	if this.Closed != that1.Closed {
		return false
	}
	if this.Synced != that1.Synced {
		return false
	}
	if !bytes.Equal(this.Data, that1.Data) {
		return false
	}
	if !bytes.Equal(this.Head, that1.Head) {
		return false
	}
	return true
}
func (this *Series) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Series)
	if !ok {
		that2, ok := that.(Series)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.UserID != that1.UserID {
		return false
	}
	if this.Fingerprint != that1.Fingerprint {
		return false
	}
	if len(this.Labels) != len(that1.Labels) {
		return false
	}
	for i := range this.Labels {
		if !this.Labels[i].Equal(that1.Labels[i]) {
			return false
		}
	}
	if len(this.Chunks) != len(that1.Chunks) {
		return false
	}
	for i := range this.Chunks {
		if !this.Chunks[i].Equal(&that1.Chunks[i]) {
			return false
		}
	}
	if !this.To.Equal(that1.To) {
		return false
	}
	if this.LastLine != that1.LastLine {
		return false
	}
	return true
}
func (this *Chunk) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 12)
	s = append(s, "&ingester.Chunk{")
	s = append(s, "From: "+fmt.Sprintf("%#v", this.From)+",\n")
	s = append(s, "To: "+fmt.Sprintf("%#v", this.To)+",\n")
	s = append(s, "FlushedAt: "+fmt.Sprintf("%#v", this.FlushedAt)+",\n")
	s = append(s, "LastUpdated: "+fmt.Sprintf("%#v", this.LastUpdated)+",\n")
	s = append(s, "Closed: "+fmt.Sprintf("%#v", this.Closed)+",\n")
	s = append(s, "Synced: "+fmt.Sprintf("%#v", this.Synced)+",\n")
	s = append(s, "Data: "+fmt.Sprintf("%#v", this.Data)+",\n")
	s = append(s, "Head: "+fmt.Sprintf("%#v", this.Head)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func (this *Series) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 10)
	s = append(s, "&ingester.Series{")
	s = append(s, "UserID: "+fmt.Sprintf("%#v", this.UserID)+",\n")
	s = append(s, "Fingerprint: "+fmt.Sprintf("%#v", this.Fingerprint)+",\n")
	s = append(s, "Labels: "+fmt.Sprintf("%#v", this.Labels)+",\n")
	if this.Chunks != nil {
		vs := make([]*Chunk, len(this.Chunks))
		for i := range vs {
			vs[i] = &this.Chunks[i]
		}
		s = append(s, "Chunks: "+fmt.Sprintf("%#v", vs)+",\n")
	}
	s = append(s, "To: "+fmt.Sprintf("%#v", this.To)+",\n")
	s = append(s, "LastLine: "+fmt.Sprintf("%#v", this.LastLine)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringCheckpoint(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *Chunk) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Chunk) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Chunk) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Head) > 0 {
		i -= len(m.Head)
		copy(dAtA[i:], m.Head)
		i = encodeVarintCheckpoint(dAtA, i, uint64(len(m.Head)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintCheckpoint(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x3a
	}
	if m.Synced {
		i--
		if m.Synced {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x30
	}
	if m.Closed {
		i--
		if m.Closed {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.LastUpdated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.LastUpdated):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintCheckpoint(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x22
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.FlushedAt, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.FlushedAt):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintCheckpoint(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x1a
	n3, err3 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.To, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.To):])
	if err3 != nil {
		return 0, err3
	}
	i -= n3
	i = encodeVarintCheckpoint(dAtA, i, uint64(n3))
	i--
	dAtA[i] = 0x12
	n4, err4 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.From, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.From):])
	if err4 != nil {
		return 0, err4
	}
	i -= n4
	i = encodeVarintCheckpoint(dAtA, i, uint64(n4))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *Series) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Series) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Series) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.LastLine) > 0 {
		i -= len(m.LastLine)
		copy(dAtA[i:], m.LastLine)
		i = encodeVarintCheckpoint(dAtA, i, uint64(len(m.LastLine)))
		i--
		dAtA[i] = 0x32
	}
	n5, err5 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.To, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.To):])
	if err5 != nil {
		return 0, err5
	}
	i -= n5
	i = encodeVarintCheckpoint(dAtA, i, uint64(n5))
	i--
	dAtA[i] = 0x2a
	if len(m.Chunks) > 0 {
		for iNdEx := len(m.Chunks) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Chunks[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintCheckpoint(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.Labels) > 0 {
		for iNdEx := len(m.Labels) - 1; iNdEx >= 0; iNdEx-- {
			{
				size := m.Labels[iNdEx].Size()
				i -= size
				if _, err := m.Labels[iNdEx].MarshalTo(dAtA[i:]); err != nil {
					return 0, err
				}
				i = encodeVarintCheckpoint(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if m.Fingerprint != 0 {
		i = encodeVarintCheckpoint(dAtA, i, uint64(m.Fingerprint))
		i--
		dAtA[i] = 0x10
	}
	if len(m.UserID) > 0 {
		i -= len(m.UserID)
		copy(dAtA[i:], m.UserID)
		i = encodeVarintCheckpoint(dAtA, i, uint64(len(m.UserID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintCheckpoint(dAtA []byte, offset int, v uint64) int {
	offset -= sovCheckpoint(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Chunk) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.From)
	n += 1 + l + sovCheckpoint(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.To)
	n += 1 + l + sovCheckpoint(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.FlushedAt)
	n += 1 + l + sovCheckpoint(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.LastUpdated)
	n += 1 + l + sovCheckpoint(uint64(l))
	if m.Closed {
		n += 2
	}
	if m.Synced {
		n += 2
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovCheckpoint(uint64(l))
	}
	l = len(m.Head)
	if l > 0 {
		n += 1 + l + sovCheckpoint(uint64(l))
	}
	return n
}

func (m *Series) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.UserID)
	if l > 0 {
		n += 1 + l + sovCheckpoint(uint64(l))
	}
	if m.Fingerprint != 0 {
		n += 1 + sovCheckpoint(uint64(m.Fingerprint))
	}
	if len(m.Labels) > 0 {
		for _, e := range m.Labels {
			l = e.Size()
			n += 1 + l + sovCheckpoint(uint64(l))
		}
	}
	if len(m.Chunks) > 0 {
		for _, e := range m.Chunks {
			l = e.Size()
			n += 1 + l + sovCheckpoint(uint64(l))
		}
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.To)
	n += 1 + l + sovCheckpoint(uint64(l))
	l = len(m.LastLine)
	if l > 0 {
		n += 1 + l + sovCheckpoint(uint64(l))
	}
	return n
}

func sovCheckpoint(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozCheckpoint(x uint64) (n int) {
	return sovCheckpoint(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *Chunk) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Chunk{`,
		`From:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.From), "Timestamp", "types.Timestamp", 1), `&`, ``, 1) + `,`,
		`To:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.To), "Timestamp", "types.Timestamp", 1), `&`, ``, 1) + `,`,
		`FlushedAt:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.FlushedAt), "Timestamp", "types.Timestamp", 1), `&`, ``, 1) + `,`,
		`LastUpdated:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.LastUpdated), "Timestamp", "types.Timestamp", 1), `&`, ``, 1) + `,`,
		`Closed:` + fmt.Sprintf("%v", this.Closed) + `,`,
		`Synced:` + fmt.Sprintf("%v", this.Synced) + `,`,
		`Data:` + fmt.Sprintf("%v", this.Data) + `,`,
		`Head:` + fmt.Sprintf("%v", this.Head) + `,`,
		`}`,
	}, "")
	return s
}
func (this *Series) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForChunks := "[]Chunk{"
	for _, f := range this.Chunks {
		repeatedStringForChunks += strings.Replace(strings.Replace(f.String(), "Chunk", "Chunk", 1), `&`, ``, 1) + ","
	}
	repeatedStringForChunks += "}"
	s := strings.Join([]string{`&Series{`,
		`UserID:` + fmt.Sprintf("%v", this.UserID) + `,`,
		`Fingerprint:` + fmt.Sprintf("%v", this.Fingerprint) + `,`,
		`Labels:` + fmt.Sprintf("%v", this.Labels) + `,`,
		`Chunks:` + repeatedStringForChunks + `,`,
		`To:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.To), "Timestamp", "types.Timestamp", 1), `&`, ``, 1) + `,`,
		`LastLine:` + fmt.Sprintf("%v", this.LastLine) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringCheckpoint(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *Chunk) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCheckpoint
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Chunk: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Chunk: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field From", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.From, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field To", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.To, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FlushedAt", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.FlushedAt, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastUpdated", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.LastUpdated, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Closed", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Closed = bool(v != 0)
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Synced", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Synced = bool(v != 0)
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Head", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Head = append(m.Head[:0], dAtA[iNdEx:postIndex]...)
			if m.Head == nil {
				m.Head = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCheckpoint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Series) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCheckpoint
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Series: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Series: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field UserID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.UserID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fingerprint", wireType)
			}
			m.Fingerprint = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Fingerprint |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Labels = append(m.Labels, github_com_cortexproject_cortex_pkg_cortexpb.LabelAdapter{})
			if err := m.Labels[len(m.Labels)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Chunks", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Chunks = append(m.Chunks, Chunk{})
			if err := m.Chunks[len(m.Chunks)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field To", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.To, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastLine", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthCheckpoint
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LastLine = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCheckpoint(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCheckpoint
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipCheckpoint(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCheckpoint
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCheckpoint
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthCheckpoint
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthCheckpoint
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowCheckpoint
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipCheckpoint(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthCheckpoint
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthCheckpoint = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCheckpoint   = fmt.Errorf("proto: integer overflow")
)
