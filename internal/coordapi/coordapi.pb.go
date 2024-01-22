// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.24.4
// source: coordapi.proto

package coordapi

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

type SetManifestRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Manifest []byte   `protobuf:"bytes,1,opt,name=Manifest,proto3" json:"Manifest,omitempty"`
	Policies [][]byte `protobuf:"bytes,2,rep,name=Policies,proto3" json:"Policies,omitempty"`
}

func (x *SetManifestRequest) Reset() {
	*x = SetManifestRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_coordapi_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetManifestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetManifestRequest) ProtoMessage() {}

func (x *SetManifestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_coordapi_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetManifestRequest.ProtoReflect.Descriptor instead.
func (*SetManifestRequest) Descriptor() ([]byte, []int) {
	return file_coordapi_proto_rawDescGZIP(), []int{0}
}

func (x *SetManifestRequest) GetManifest() []byte {
	if x != nil {
		return x.Manifest
	}
	return nil
}

func (x *SetManifestRequest) GetPolicies() [][]byte {
	if x != nil {
		return x.Policies
	}
	return nil
}

type SetManifestResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// PEM-encoded certificate
	CACert []byte `protobuf:"bytes,1,opt,name=CACert,proto3" json:"CACert,omitempty"`
	// PEM-encoded certificate
	IntermCert []byte `protobuf:"bytes,2,opt,name=IntermCert,proto3" json:"IntermCert,omitempty"`
}

func (x *SetManifestResponse) Reset() {
	*x = SetManifestResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_coordapi_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetManifestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetManifestResponse) ProtoMessage() {}

func (x *SetManifestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_coordapi_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetManifestResponse.ProtoReflect.Descriptor instead.
func (*SetManifestResponse) Descriptor() ([]byte, []int) {
	return file_coordapi_proto_rawDescGZIP(), []int{1}
}

func (x *SetManifestResponse) GetCACert() []byte {
	if x != nil {
		return x.CACert
	}
	return nil
}

func (x *SetManifestResponse) GetIntermCert() []byte {
	if x != nil {
		return x.IntermCert
	}
	return nil
}

type GetManifestsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetManifestsRequest) Reset() {
	*x = GetManifestsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_coordapi_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetManifestsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetManifestsRequest) ProtoMessage() {}

func (x *GetManifestsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_coordapi_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetManifestsRequest.ProtoReflect.Descriptor instead.
func (*GetManifestsRequest) Descriptor() ([]byte, []int) {
	return file_coordapi_proto_rawDescGZIP(), []int{2}
}

type GetManifestsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Manifests [][]byte `protobuf:"bytes,1,rep,name=Manifests,proto3" json:"Manifests,omitempty"`
	Policies  [][]byte `protobuf:"bytes,2,rep,name=Policies,proto3" json:"Policies,omitempty"`
	// PEM-encoded certificate
	CACert []byte `protobuf:"bytes,3,opt,name=CACert,proto3" json:"CACert,omitempty"`
	// PEM-encoded certificate
	IntermCert []byte `protobuf:"bytes,4,opt,name=IntermCert,proto3" json:"IntermCert,omitempty"`
}

func (x *GetManifestsResponse) Reset() {
	*x = GetManifestsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_coordapi_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetManifestsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetManifestsResponse) ProtoMessage() {}

func (x *GetManifestsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_coordapi_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetManifestsResponse.ProtoReflect.Descriptor instead.
func (*GetManifestsResponse) Descriptor() ([]byte, []int) {
	return file_coordapi_proto_rawDescGZIP(), []int{3}
}

func (x *GetManifestsResponse) GetManifests() [][]byte {
	if x != nil {
		return x.Manifests
	}
	return nil
}

func (x *GetManifestsResponse) GetPolicies() [][]byte {
	if x != nil {
		return x.Policies
	}
	return nil
}

func (x *GetManifestsResponse) GetCACert() []byte {
	if x != nil {
		return x.CACert
	}
	return nil
}

func (x *GetManifestsResponse) GetIntermCert() []byte {
	if x != nil {
		return x.IntermCert
	}
	return nil
}

var File_coordapi_proto protoreflect.FileDescriptor

var file_coordapi_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x08, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69, 0x22, 0x4c, 0x0a, 0x12, 0x53, 0x65,
	0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1a, 0x0a, 0x08, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x08, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x08,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x4d, 0x0a, 0x13, 0x53, 0x65, 0x74, 0x4d,
	0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x06, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x49, 0x6e, 0x74, 0x65, 0x72,
	0x6d, 0x43, 0x65, 0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x22, 0x15, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x4d, 0x61,
	0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x88,
	0x01, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x4d, 0x61, 0x6e, 0x69, 0x66,
	0x65, 0x73, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x09, 0x4d, 0x61, 0x6e, 0x69,
	0x66, 0x65, 0x73, 0x74, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65,
	0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x08, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65,
	0x73, 0x12, 0x16, 0x0a, 0x06, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x06, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x49,
	0x6e, 0x74, 0x65, 0x72, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x32, 0xa5, 0x01, 0x0a, 0x08, 0x43, 0x6f,
	0x6f, 0x72, 0x64, 0x41, 0x50, 0x49, 0x12, 0x4a, 0x0a, 0x0b, 0x53, 0x65, 0x74, 0x4d, 0x61, 0x6e,
	0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x2e, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69,
	0x2e, 0x53, 0x65, 0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69, 0x2e, 0x53,
	0x65, 0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x4d, 0x0a, 0x0c, 0x47, 0x65, 0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73,
	0x74, 0x73, 0x12, 0x1d, 0x2e, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x65,
	0x74, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1e, 0x2e, 0x63, 0x6f, 0x6f, 0x72, 0x64, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x65, 0x74,
	0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x30, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x65, 0x64, 0x67, 0x65, 0x6c, 0x65, 0x73, 0x73, 0x73, 0x79, 0x73, 0x2f, 0x6e, 0x75, 0x6e, 0x6b,
	0x69, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6f, 0x6f, 0x72, 0x64,
	0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_coordapi_proto_rawDescOnce sync.Once
	file_coordapi_proto_rawDescData = file_coordapi_proto_rawDesc
)

func file_coordapi_proto_rawDescGZIP() []byte {
	file_coordapi_proto_rawDescOnce.Do(func() {
		file_coordapi_proto_rawDescData = protoimpl.X.CompressGZIP(file_coordapi_proto_rawDescData)
	})
	return file_coordapi_proto_rawDescData
}

var file_coordapi_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_coordapi_proto_goTypes = []interface{}{
	(*SetManifestRequest)(nil),   // 0: coordapi.SetManifestRequest
	(*SetManifestResponse)(nil),  // 1: coordapi.SetManifestResponse
	(*GetManifestsRequest)(nil),  // 2: coordapi.GetManifestsRequest
	(*GetManifestsResponse)(nil), // 3: coordapi.GetManifestsResponse
}
var file_coordapi_proto_depIdxs = []int32{
	0, // 0: coordapi.CoordAPI.SetManifest:input_type -> coordapi.SetManifestRequest
	2, // 1: coordapi.CoordAPI.GetManifests:input_type -> coordapi.GetManifestsRequest
	1, // 2: coordapi.CoordAPI.SetManifest:output_type -> coordapi.SetManifestResponse
	3, // 3: coordapi.CoordAPI.GetManifests:output_type -> coordapi.GetManifestsResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_coordapi_proto_init() }
func file_coordapi_proto_init() {
	if File_coordapi_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_coordapi_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetManifestRequest); i {
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
		file_coordapi_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetManifestResponse); i {
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
		file_coordapi_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetManifestsRequest); i {
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
		file_coordapi_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetManifestsResponse); i {
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
			RawDescriptor: file_coordapi_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_coordapi_proto_goTypes,
		DependencyIndexes: file_coordapi_proto_depIdxs,
		MessageInfos:      file_coordapi_proto_msgTypes,
	}.Build()
	File_coordapi_proto = out.File
	file_coordapi_proto_rawDesc = nil
	file_coordapi_proto_goTypes = nil
	file_coordapi_proto_depIdxs = nil
}
