// Copyright 2015 gRPC authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v3.21.12
// source: src/message/message.proto

package message

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PodRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Node          string                 `protobuf:"bytes,1,opt,name=node,proto3" json:"node,omitempty"`
	Signal        float64                `protobuf:"fixed64,2,opt,name=signal,proto3" json:"signal,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PodRequest) Reset() {
	*x = PodRequest{}
	mi := &file_src_message_message_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PodRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PodRequest) ProtoMessage() {}

func (x *PodRequest) ProtoReflect() protoreflect.Message {
	mi := &file_src_message_message_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PodRequest.ProtoReflect.Descriptor instead.
func (*PodRequest) Descriptor() ([]byte, []int) {
	return file_src_message_message_proto_rawDescGZIP(), []int{0}
}

func (x *PodRequest) GetNode() string {
	if x != nil {
		return x.Node
	}
	return ""
}

func (x *PodRequest) GetSignal() float64 {
	if x != nil {
		return x.Signal
	}
	return 0
}

type EmptyReply struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EmptyReply) Reset() {
	*x = EmptyReply{}
	mi := &file_src_message_message_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EmptyReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmptyReply) ProtoMessage() {}

func (x *EmptyReply) ProtoReflect() protoreflect.Message {
	mi := &file_src_message_message_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmptyReply.ProtoReflect.Descriptor instead.
func (*EmptyReply) Descriptor() ([]byte, []int) {
	return file_src_message_message_proto_rawDescGZIP(), []int{1}
}

type DenseMatrix struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Rows          int64                  `protobuf:"varint,1,opt,name=rows,proto3" json:"rows,omitempty"`
	Cols          int64                  `protobuf:"varint,2,opt,name=cols,proto3" json:"cols,omitempty"`
	Data          []float64              `protobuf:"fixed64,3,rep,packed,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DenseMatrix) Reset() {
	*x = DenseMatrix{}
	mi := &file_src_message_message_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DenseMatrix) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DenseMatrix) ProtoMessage() {}

func (x *DenseMatrix) ProtoReflect() protoreflect.Message {
	mi := &file_src_message_message_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DenseMatrix.ProtoReflect.Descriptor instead.
func (*DenseMatrix) Descriptor() ([]byte, []int) {
	return file_src_message_message_proto_rawDescGZIP(), []int{2}
}

func (x *DenseMatrix) GetRows() int64 {
	if x != nil {
		return x.Rows
	}
	return 0
}

func (x *DenseMatrix) GetCols() int64 {
	if x != nil {
		return x.Cols
	}
	return 0
}

func (x *DenseMatrix) GetData() []float64 {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_src_message_message_proto protoreflect.FileDescriptor

var file_src_message_message_proto_rawDesc = string([]byte{
	0x0a, 0x19, 0x73, 0x72, 0x63, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2f, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x22, 0x38, 0x0a, 0x0a, 0x50, 0x6f, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x01, 0x52, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x22, 0x0c,
	0x0a, 0x0a, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x4d, 0x0a, 0x0b,
	0x44, 0x65, 0x6e, 0x73, 0x65, 0x4d, 0x61, 0x74, 0x72, 0x69, 0x78, 0x12, 0x12, 0x0a, 0x04, 0x72,
	0x6f, 0x77, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x72, 0x6f, 0x77, 0x73, 0x12,
	0x12, 0x0a, 0x04, 0x63, 0x6f, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x63,
	0x6f, 0x6c, 0x73, 0x12, 0x16, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x01, 0x42, 0x02, 0x10, 0x01, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x32, 0x46, 0x0a, 0x0c, 0x50,
	0x6f, 0x64, 0x50, 0x6c, 0x61, 0x63, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x36, 0x0a, 0x0a, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x50, 0x6f, 0x64, 0x12, 0x13, 0x2e, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x2e, 0x50, 0x6f, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x13,
	0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x52, 0x65,
	0x70, 0x6c, 0x79, 0x32, 0x4f, 0x0a, 0x0e, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x65,
	0x4d, 0x65, 0x72, 0x67, 0x65, 0x12, 0x3d, 0x0a, 0x0f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x41, 0x67, 0x67, 0x4d, 0x65, 0x72, 0x67, 0x65, 0x12, 0x14, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x2e, 0x44, 0x65, 0x6e, 0x73, 0x65, 0x4d, 0x61, 0x74, 0x72, 0x69, 0x78, 0x1a, 0x14,
	0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x44, 0x65, 0x6e, 0x73, 0x65, 0x4d, 0x61,
	0x74, 0x72, 0x69, 0x78, 0x42, 0x28, 0x5a, 0x26, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x4c, 0x75, 0x63, 0x61, 0x43, 0x68, 0x6f, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x6e,
	0x74, 0x6f, 0x2f, 0x73, 0x72, 0x63, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_src_message_message_proto_rawDescOnce sync.Once
	file_src_message_message_proto_rawDescData []byte
)

func file_src_message_message_proto_rawDescGZIP() []byte {
	file_src_message_message_proto_rawDescOnce.Do(func() {
		file_src_message_message_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_src_message_message_proto_rawDesc), len(file_src_message_message_proto_rawDesc)))
	})
	return file_src_message_message_proto_rawDescData
}

var file_src_message_message_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_src_message_message_proto_goTypes = []any{
	(*PodRequest)(nil),  // 0: message.PodRequest
	(*EmptyReply)(nil),  // 1: message.EmptyReply
	(*DenseMatrix)(nil), // 2: message.DenseMatrix
}
var file_src_message_message_proto_depIdxs = []int32{
	0, // 0: message.PodPlacement.RequestPod:input_type -> message.PodRequest
	2, // 1: message.AggregateMerge.RequestAggMerge:input_type -> message.DenseMatrix
	1, // 2: message.PodPlacement.RequestPod:output_type -> message.EmptyReply
	2, // 3: message.AggregateMerge.RequestAggMerge:output_type -> message.DenseMatrix
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_src_message_message_proto_init() }
func file_src_message_message_proto_init() {
	if File_src_message_message_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_src_message_message_proto_rawDesc), len(file_src_message_message_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_src_message_message_proto_goTypes,
		DependencyIndexes: file_src_message_message_proto_depIdxs,
		MessageInfos:      file_src_message_message_proto_msgTypes,
	}.Build()
	File_src_message_message_proto = out.File
	file_src_message_message_proto_goTypes = nil
	file_src_message_message_proto_depIdxs = nil
}
