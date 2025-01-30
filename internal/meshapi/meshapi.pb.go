// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        v5.29.2
// source: meshapi.proto

package meshapi

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

type NewMeshCertRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NewMeshCertRequest) Reset() {
	*x = NewMeshCertRequest{}
	mi := &file_meshapi_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NewMeshCertRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewMeshCertRequest) ProtoMessage() {}

func (x *NewMeshCertRequest) ProtoReflect() protoreflect.Message {
	mi := &file_meshapi_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewMeshCertRequest.ProtoReflect.Descriptor instead.
func (*NewMeshCertRequest) Descriptor() ([]byte, []int) {
	return file_meshapi_proto_rawDescGZIP(), []int{0}
}

type NewMeshCertResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// PEM-encoded certificate used by the workload as CA
	MeshCACert []byte `protobuf:"bytes,1,opt,name=MeshCACert,proto3" json:"MeshCACert,omitempty"`
	// Concatenated PEM-encoded certificates used by the workload certificate chain
	CertChain []byte `protobuf:"bytes,2,opt,name=CertChain,proto3" json:"CertChain,omitempty"`
	// PEM-encoded certificate when workloads trust also workloads from previous manifests
	RootCACert []byte `protobuf:"bytes,3,opt,name=RootCACert,proto3" json:"RootCACert,omitempty"`
	// Raw byte slice which can be used to derive more secrets
	WorkloadSecret []byte `protobuf:"bytes,4,opt,name=WorkloadSecret,proto3" json:"WorkloadSecret,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *NewMeshCertResponse) Reset() {
	*x = NewMeshCertResponse{}
	mi := &file_meshapi_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NewMeshCertResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewMeshCertResponse) ProtoMessage() {}

func (x *NewMeshCertResponse) ProtoReflect() protoreflect.Message {
	mi := &file_meshapi_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewMeshCertResponse.ProtoReflect.Descriptor instead.
func (*NewMeshCertResponse) Descriptor() ([]byte, []int) {
	return file_meshapi_proto_rawDescGZIP(), []int{1}
}

func (x *NewMeshCertResponse) GetMeshCACert() []byte {
	if x != nil {
		return x.MeshCACert
	}
	return nil
}

func (x *NewMeshCertResponse) GetCertChain() []byte {
	if x != nil {
		return x.CertChain
	}
	return nil
}

func (x *NewMeshCertResponse) GetRootCACert() []byte {
	if x != nil {
		return x.RootCACert
	}
	return nil
}

func (x *NewMeshCertResponse) GetWorkloadSecret() []byte {
	if x != nil {
		return x.WorkloadSecret
	}
	return nil
}

var File_meshapi_proto protoreflect.FileDescriptor

var file_meshapi_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x6d, 0x65, 0x73, 0x68, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x6d, 0x65, 0x73, 0x68, 0x61, 0x70, 0x69, 0x22, 0x2d, 0x0a, 0x12, 0x4e, 0x65, 0x77, 0x4d,
	0x65, 0x73, 0x68, 0x43, 0x65, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x4a, 0x04,
	0x08, 0x01, 0x10, 0x02, 0x52, 0x11, 0x50, 0x65, 0x65, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x4b, 0x65, 0x79, 0x48, 0x61, 0x73, 0x68, 0x22, 0x9b, 0x01, 0x0a, 0x13, 0x4e, 0x65, 0x77, 0x4d,
	0x65, 0x73, 0x68, 0x43, 0x65, 0x72, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x1e, 0x0a, 0x0a, 0x4d, 0x65, 0x73, 0x68, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x0a, 0x4d, 0x65, 0x73, 0x68, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x12,
	0x1c, 0x0a, 0x09, 0x43, 0x65, 0x72, 0x74, 0x43, 0x68, 0x61, 0x69, 0x6e, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x09, 0x43, 0x65, 0x72, 0x74, 0x43, 0x68, 0x61, 0x69, 0x6e, 0x12, 0x1e, 0x0a,
	0x0a, 0x52, 0x6f, 0x6f, 0x74, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x0a, 0x52, 0x6f, 0x6f, 0x74, 0x43, 0x41, 0x43, 0x65, 0x72, 0x74, 0x12, 0x26, 0x0a,
	0x0e, 0x57, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0e, 0x57, 0x6f, 0x72, 0x6b, 0x6c, 0x6f, 0x61, 0x64, 0x53,
	0x65, 0x63, 0x72, 0x65, 0x74, 0x32, 0x53, 0x0a, 0x07, 0x4d, 0x65, 0x73, 0x68, 0x41, 0x50, 0x49,
	0x12, 0x48, 0x0a, 0x0b, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x73, 0x68, 0x43, 0x65, 0x72, 0x74, 0x12,
	0x1b, 0x2e, 0x6d, 0x65, 0x73, 0x68, 0x61, 0x70, 0x69, 0x2e, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x73,
	0x68, 0x43, 0x65, 0x72, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x6d,
	0x65, 0x73, 0x68, 0x61, 0x70, 0x69, 0x2e, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x73, 0x68, 0x43, 0x65,
	0x72, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x32, 0x5a, 0x30, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x65, 0x64, 0x67, 0x65, 0x6c, 0x65, 0x73,
	0x73, 0x73, 0x79, 0x73, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x73, 0x74, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d, 0x65, 0x73, 0x68, 0x61, 0x70, 0x69, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_meshapi_proto_rawDescOnce sync.Once
	file_meshapi_proto_rawDescData = file_meshapi_proto_rawDesc
)

func file_meshapi_proto_rawDescGZIP() []byte {
	file_meshapi_proto_rawDescOnce.Do(func() {
		file_meshapi_proto_rawDescData = protoimpl.X.CompressGZIP(file_meshapi_proto_rawDescData)
	})
	return file_meshapi_proto_rawDescData
}

var file_meshapi_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_meshapi_proto_goTypes = []any{
	(*NewMeshCertRequest)(nil),  // 0: meshapi.NewMeshCertRequest
	(*NewMeshCertResponse)(nil), // 1: meshapi.NewMeshCertResponse
}
var file_meshapi_proto_depIdxs = []int32{
	0, // 0: meshapi.MeshAPI.NewMeshCert:input_type -> meshapi.NewMeshCertRequest
	1, // 1: meshapi.MeshAPI.NewMeshCert:output_type -> meshapi.NewMeshCertResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_meshapi_proto_init() }
func file_meshapi_proto_init() {
	if File_meshapi_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_meshapi_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_meshapi_proto_goTypes,
		DependencyIndexes: file_meshapi_proto_depIdxs,
		MessageInfos:      file_meshapi_proto_msgTypes,
	}.Build()
	File_meshapi_proto = out.File
	file_meshapi_proto_rawDesc = nil
	file_meshapi_proto_goTypes = nil
	file_meshapi_proto_depIdxs = nil
}
