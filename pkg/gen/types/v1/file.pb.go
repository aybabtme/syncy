// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        (unknown)
// source: types/v1/file.proto

package typesv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Info    *FileInfo `protobuf:"bytes,1,opt,name=info,proto3" json:"info,omitempty"`
	Content []byte    `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_v1_file_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_types_v1_file_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use File.ProtoReflect.Descriptor instead.
func (*File) Descriptor() ([]byte, []int) {
	return file_types_v1_file_proto_rawDescGZIP(), []int{0}
}

func (x *File) GetInfo() *FileInfo {
	if x != nil {
		return x.Info
	}
	return nil
}

func (x *File) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

type FileMode struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mode uint32 `protobuf:"varint,1,opt,name=mode,proto3" json:"mode,omitempty"`
}

func (x *FileMode) Reset() {
	*x = FileMode{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_v1_file_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileMode) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileMode) ProtoMessage() {}

func (x *FileMode) ProtoReflect() protoreflect.Message {
	mi := &file_types_v1_file_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileMode.ProtoReflect.Descriptor instead.
func (*FileMode) Descriptor() ([]byte, []int) {
	return file_types_v1_file_proto_rawDescGZIP(), []int{1}
}

func (x *FileMode) GetMode() uint32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

type FileInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path     *Path                  `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Name     string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Size     uint64                 `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	Mode     *FileMode              `protobuf:"bytes,4,opt,name=mode,proto3" json:"mode,omitempty"`
	ModeTime *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=mode_time,json=modeTime,proto3" json:"mode_time,omitempty"`
	IsDir    bool                   `protobuf:"varint,6,opt,name=is_dir,json=isDir,proto3" json:"is_dir,omitempty"`
}

func (x *FileInfo) Reset() {
	*x = FileInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_types_v1_file_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileInfo) ProtoMessage() {}

func (x *FileInfo) ProtoReflect() protoreflect.Message {
	mi := &file_types_v1_file_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileInfo.ProtoReflect.Descriptor instead.
func (*FileInfo) Descriptor() ([]byte, []int) {
	return file_types_v1_file_proto_rawDescGZIP(), []int{2}
}

func (x *FileInfo) GetPath() *Path {
	if x != nil {
		return x.Path
	}
	return nil
}

func (x *FileInfo) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FileInfo) GetSize() uint64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *FileInfo) GetMode() *FileMode {
	if x != nil {
		return x.Mode
	}
	return nil
}

func (x *FileInfo) GetModeTime() *timestamppb.Timestamp {
	if x != nil {
		return x.ModeTime
	}
	return nil
}

func (x *FileInfo) GetIsDir() bool {
	if x != nil {
		return x.IsDir
	}
	return false
}

var File_types_v1_file_proto protoreflect.FileDescriptor

var file_types_v1_file_proto_rawDesc = []byte{
	0x0a, 0x13, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x1a,
	0x13, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x61, 0x74, 0x68, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x48, 0x0a, 0x04, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x26, 0x0a,
	0x04, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x74, 0x79,
	0x70, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52,
	0x04, 0x69, 0x6e, 0x66, 0x6f, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22,
	0x1e, 0x0a, 0x08, 0x46, 0x69, 0x6c, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6d,
	0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x6d, 0x6f, 0x64, 0x65, 0x22,
	0xce, 0x01, 0x0a, 0x08, 0x46, 0x69, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x22, 0x0a, 0x04,
	0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x74, 0x68, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x26, 0x0a, 0x04, 0x6d, 0x6f, 0x64, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x76,
	0x31, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x52, 0x04, 0x6d, 0x6f, 0x64, 0x65,
	0x12, 0x37, 0x0a, 0x09, 0x6d, 0x6f, 0x64, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52,
	0x08, 0x6d, 0x6f, 0x64, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x15, 0x0a, 0x06, 0x69, 0x73, 0x5f,
	0x64, 0x69, 0x72, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x69, 0x73, 0x44, 0x69, 0x72,
	0x42, 0x8e, 0x01, 0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x76,
	0x31, 0x42, 0x09, 0x46, 0x69, 0x6c, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x32,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x79, 0x62, 0x61, 0x62,
	0x74, 0x6d, 0x65, 0x2f, 0x73, 0x79, 0x6e, 0x63, 0x79, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x65,
	0x6e, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x76, 0x31, 0x3b, 0x74, 0x79, 0x70, 0x65, 0x73,
	0x76, 0x31, 0xa2, 0x02, 0x03, 0x54, 0x58, 0x58, 0xaa, 0x02, 0x08, 0x54, 0x79, 0x70, 0x65, 0x73,
	0x2e, 0x56, 0x31, 0xca, 0x02, 0x08, 0x54, 0x79, 0x70, 0x65, 0x73, 0x5c, 0x56, 0x31, 0xe2, 0x02,
	0x14, 0x54, 0x79, 0x70, 0x65, 0x73, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x09, 0x54, 0x79, 0x70, 0x65, 0x73, 0x3a, 0x3a, 0x56,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_types_v1_file_proto_rawDescOnce sync.Once
	file_types_v1_file_proto_rawDescData = file_types_v1_file_proto_rawDesc
)

func file_types_v1_file_proto_rawDescGZIP() []byte {
	file_types_v1_file_proto_rawDescOnce.Do(func() {
		file_types_v1_file_proto_rawDescData = protoimpl.X.CompressGZIP(file_types_v1_file_proto_rawDescData)
	})
	return file_types_v1_file_proto_rawDescData
}

var file_types_v1_file_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_types_v1_file_proto_goTypes = []interface{}{
	(*File)(nil),                  // 0: types.v1.File
	(*FileMode)(nil),              // 1: types.v1.FileMode
	(*FileInfo)(nil),              // 2: types.v1.FileInfo
	(*Path)(nil),                  // 3: types.v1.Path
	(*timestamppb.Timestamp)(nil), // 4: google.protobuf.Timestamp
}
var file_types_v1_file_proto_depIdxs = []int32{
	2, // 0: types.v1.File.info:type_name -> types.v1.FileInfo
	3, // 1: types.v1.FileInfo.path:type_name -> types.v1.Path
	1, // 2: types.v1.FileInfo.mode:type_name -> types.v1.FileMode
	4, // 3: types.v1.FileInfo.mode_time:type_name -> google.protobuf.Timestamp
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_types_v1_file_proto_init() }
func file_types_v1_file_proto_init() {
	if File_types_v1_file_proto != nil {
		return
	}
	file_types_v1_path_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_types_v1_file_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*File); i {
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
		file_types_v1_file_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileMode); i {
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
		file_types_v1_file_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileInfo); i {
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
			RawDescriptor: file_types_v1_file_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_types_v1_file_proto_goTypes,
		DependencyIndexes: file_types_v1_file_proto_depIdxs,
		MessageInfos:      file_types_v1_file_proto_msgTypes,
	}.Build()
	File_types_v1_file_proto = out.File
	file_types_v1_file_proto_rawDesc = nil
	file_types_v1_file_proto_goTypes = nil
	file_types_v1_file_proto_depIdxs = nil
}