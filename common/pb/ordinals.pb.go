// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v3.12.4
// source: common/pb/ordinals.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MyRange struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Start int64 `protobuf:"varint,1,opt,name=start,proto3" json:"start,omitempty"`
	Size  int64 `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
}

func (x *MyRange) Reset() {
	*x = MyRange{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_pb_ordinals_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MyRange) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MyRange) ProtoMessage() {}

func (x *MyRange) ProtoReflect() protoreflect.Message {
	mi := &file_common_pb_ordinals_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MyRange.ProtoReflect.Descriptor instead.
func (*MyRange) Descriptor() ([]byte, []int) {
	return file_common_pb_ordinals_proto_rawDescGZIP(), []int{0}
}

func (x *MyRange) GetStart() int64 {
	if x != nil {
		return x.Start
	}
	return 0
}

func (x *MyRange) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

type MyUtxoValueInDB struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UtxoId      uint64     `protobuf:"varint,1,opt,name=utxo_id,json=utxoId,proto3" json:"utxo_id,omitempty"`
	AddressType uint32     `protobuf:"varint,2,opt,name=address_type,json=addressType,proto3" json:"address_type,omitempty"`
	ReqSigs     uint32     `protobuf:"varint,3,opt,name=reqSigs,proto3" json:"reqSigs,omitempty"`
	AddressIds  []uint64   `protobuf:"varint,4,rep,packed,name=address_ids,json=addressIds,proto3" json:"address_ids,omitempty"`
	Ordinals    []*MyRange `protobuf:"bytes,5,rep,name=ordinals,proto3" json:"ordinals,omitempty"`
}

func (x *MyUtxoValueInDB) Reset() {
	*x = MyUtxoValueInDB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_pb_ordinals_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MyUtxoValueInDB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MyUtxoValueInDB) ProtoMessage() {}

func (x *MyUtxoValueInDB) ProtoReflect() protoreflect.Message {
	mi := &file_common_pb_ordinals_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MyUtxoValueInDB.ProtoReflect.Descriptor instead.
func (*MyUtxoValueInDB) Descriptor() ([]byte, []int) {
	return file_common_pb_ordinals_proto_rawDescGZIP(), []int{1}
}

func (x *MyUtxoValueInDB) GetUtxoId() uint64 {
	if x != nil {
		return x.UtxoId
	}
	return 0
}

func (x *MyUtxoValueInDB) GetAddressType() uint32 {
	if x != nil {
		return x.AddressType
	}
	return 0
}

func (x *MyUtxoValueInDB) GetReqSigs() uint32 {
	if x != nil {
		return x.ReqSigs
	}
	return 0
}

func (x *MyUtxoValueInDB) GetAddressIds() []uint64 {
	if x != nil {
		return x.AddressIds
	}
	return nil
}

func (x *MyUtxoValueInDB) GetOrdinals() []*MyRange {
	if x != nil {
		return x.Ordinals
	}
	return nil
}

var File_common_pb_ordinals_proto protoreflect.FileDescriptor

var file_common_pb_ordinals_proto_rawDesc = []byte{
	0x0a, 0x18, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x70, 0x62, 0x2f, 0x6f, 0x72, 0x64, 0x69,
	0x6e, 0x61, 0x6c, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x70, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x22, 0x33, 0x0a, 0x07, 0x4d, 0x79, 0x52, 0x61, 0x6e, 0x67, 0x65,
	0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x22, 0xb8, 0x01, 0x0a, 0x0f, 0x4d,
	0x79, 0x55, 0x74, 0x78, 0x6f, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x49, 0x6e, 0x44, 0x42, 0x12, 0x17,
	0x0a, 0x07, 0x75, 0x74, 0x78, 0x6f, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x06, 0x75, 0x74, 0x78, 0x6f, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0b, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x54, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x65,
	0x71, 0x53, 0x69, 0x67, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x72, 0x65, 0x71,
	0x53, 0x69, 0x67, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x5f,
	0x69, 0x64, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x04, 0x52, 0x0a, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x49, 0x64, 0x73, 0x12, 0x2e, 0x0a, 0x08, 0x6f, 0x72, 0x64, 0x69, 0x6e, 0x61, 0x6c,
	0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x6d, 0x6f, 0x6e, 0x2e, 0x4d, 0x79, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x08, 0x6f, 0x72, 0x64,
	0x69, 0x6e, 0x61, 0x6c, 0x73, 0x42, 0x0c, 0x5a, 0x0a, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e,
	0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_common_pb_ordinals_proto_rawDescOnce sync.Once
	file_common_pb_ordinals_proto_rawDescData = file_common_pb_ordinals_proto_rawDesc
)

func file_common_pb_ordinals_proto_rawDescGZIP() []byte {
	file_common_pb_ordinals_proto_rawDescOnce.Do(func() {
		file_common_pb_ordinals_proto_rawDescData = protoimpl.X.CompressGZIP(file_common_pb_ordinals_proto_rawDescData)
	})
	return file_common_pb_ordinals_proto_rawDescData
}

var file_common_pb_ordinals_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_common_pb_ordinals_proto_goTypes = []any{
	(*MyRange)(nil),         // 0: pb.common.MyRange
	(*MyUtxoValueInDB)(nil), // 1: pb.common.MyUtxoValueInDB
}
var file_common_pb_ordinals_proto_depIdxs = []int32{
	0, // 0: pb.common.MyUtxoValueInDB.ordinals:type_name -> pb.common.MyRange
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_common_pb_ordinals_proto_init() }
func file_common_pb_ordinals_proto_init() {
	if File_common_pb_ordinals_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_common_pb_ordinals_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*MyRange); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_pb_ordinals_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*MyUtxoValueInDB); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_common_pb_ordinals_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_pb_ordinals_proto_goTypes,
		DependencyIndexes: file_common_pb_ordinals_proto_depIdxs,
		MessageInfos:      file_common_pb_ordinals_proto_msgTypes,
	}.Build()
	File_common_pb_ordinals_proto = out.File
	file_common_pb_ordinals_proto_rawDesc = nil
	file_common_pb_ordinals_proto_goTypes = nil
	file_common_pb_ordinals_proto_depIdxs = nil
}
