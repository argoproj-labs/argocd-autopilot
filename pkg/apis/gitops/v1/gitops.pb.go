// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.15.2
// source: pkg/apis/gitops/v1/gitops.proto

// Gitops Service
//
// Gitops Service API allows you to perform CRUD operations on manifests
// in a specified git repository.

package v1

import (
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/genproto/googleapis/rpc/status"
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

// Repository represents a remote git repository
type Repository struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Provider string `protobuf:"bytes,1,opt,name=provider,proto3" json:"provider,omitempty"`
	CloneUrl string `protobuf:"bytes,2,opt,name=clone_url,json=cloneUrl,proto3" json:"clone_url,omitempty"`
}

func (x *Repository) Reset() {
	*x = Repository{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Repository) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Repository) ProtoMessage() {}

func (x *Repository) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Repository.ProtoReflect.Descriptor instead.
func (*Repository) Descriptor() ([]byte, []int) {
	return file_pkg_apis_gitops_v1_gitops_proto_rawDescGZIP(), []int{0}
}

func (x *Repository) GetProvider() string {
	if x != nil {
		return x.Provider
	}
	return ""
}

func (x *Repository) GetCloneUrl() string {
	if x != nil {
		return x.CloneUrl
	}
	return ""
}

// Manifest represents a file containing a manifest
type Manifest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Data string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Manifest) Reset() {
	*x = Manifest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Manifest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Manifest) ProtoMessage() {}

func (x *Manifest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Manifest.ProtoReflect.Descriptor instead.
func (*Manifest) Descriptor() ([]byte, []int) {
	return file_pkg_apis_gitops_v1_gitops_proto_rawDescGZIP(), []int{1}
}

func (x *Manifest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Manifest) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

type AddManifestRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Repo     *Repository `protobuf:"bytes,1,opt,name=repo,proto3" json:"repo,omitempty"`
	Manifest *Manifest   `protobuf:"bytes,2,opt,name=manifest,proto3" json:"manifest,omitempty"`
}

func (x *AddManifestRequest) Reset() {
	*x = AddManifestRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddManifestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddManifestRequest) ProtoMessage() {}

func (x *AddManifestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddManifestRequest.ProtoReflect.Descriptor instead.
func (*AddManifestRequest) Descriptor() ([]byte, []int) {
	return file_pkg_apis_gitops_v1_gitops_proto_rawDescGZIP(), []int{2}
}

func (x *AddManifestRequest) GetRepo() *Repository {
	if x != nil {
		return x.Repo
	}
	return nil
}

func (x *AddManifestRequest) GetManifest() *Manifest {
	if x != nil {
		return x.Manifest
	}
	return nil
}

type AddManifestResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path string `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
}

func (x *AddManifestResponse) Reset() {
	*x = AddManifestResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddManifestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddManifestResponse) ProtoMessage() {}

func (x *AddManifestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_apis_gitops_v1_gitops_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddManifestResponse.ProtoReflect.Descriptor instead.
func (*AddManifestResponse) Descriptor() ([]byte, []int) {
	return file_pkg_apis_gitops_v1_gitops_proto_rawDescGZIP(), []int{3}
}

func (x *AddManifestResponse) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

var File_pkg_apis_gitops_v1_gitops_proto protoreflect.FileDescriptor

var file_pkg_apis_gitops_v1_gitops_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x67, 0x69, 0x74, 0x6f, 0x70,
	0x73, 0x2f, 0x76, 0x31, 0x2f, 0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x72,
	0x67, 0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2e, 0x61, 0x72, 0x67, 0x6f, 0x63, 0x64, 0x5f, 0x61, 0x75,
	0x74, 0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x61, 0x70, 0x69, 0x73,
	0x2e, 0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x76, 0x31, 0x1a, 0x17, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f,
	0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67, 0x65, 0x6e, 0x2d, 0x6f, 0x70,
	0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2f,
	0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x97, 0x01, 0x0a, 0x0a, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79,
	0x12, 0x45, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x42, 0x29, 0x92, 0x41, 0x26, 0x32, 0x11, 0x67, 0x69, 0x74, 0x20, 0x70, 0x72, 0x6f,
	0x76, 0x69, 0x64, 0x65, 0x72, 0x20, 0x6e, 0x61, 0x6d, 0x65, 0x3a, 0x06, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0xd2, 0x01, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x52, 0x08, 0x70,
	0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x12, 0x42, 0x0a, 0x09, 0x63, 0x6c, 0x6f, 0x6e, 0x65,
	0x5f, 0x75, 0x72, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x25, 0x92, 0x41, 0x22, 0x32,
	0x14, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x20, 0x63, 0x6c, 0x6f, 0x6e,
	0x65, 0x20, 0x75, 0x72, 0x6c, 0xd2, 0x01, 0x09, 0x63, 0x6c, 0x6f, 0x6e, 0x65, 0x5f, 0x75, 0x72,
	0x6c, 0x52, 0x08, 0x63, 0x6c, 0x6f, 0x6e, 0x65, 0x55, 0x72, 0x6c, 0x22, 0x6d, 0x0a, 0x08, 0x4d,
	0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x32, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1e, 0x92, 0x41, 0x1b, 0x32, 0x12, 0x6d, 0x61, 0x6e, 0x69,
	0x66, 0x65, 0x73, 0x74, 0x20, 0x66, 0x69, 0x6c, 0x65, 0x20, 0x6e, 0x61, 0x6d, 0x65, 0xd2, 0x01,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2d, 0x0a, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x19, 0x92, 0x41, 0x16, 0x32, 0x0d,
	0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x20, 0x64, 0x61, 0x74, 0x61, 0xd2, 0x01, 0x04,
	0x64, 0x61, 0x74, 0x61, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x8c, 0x02, 0x0a, 0x12, 0x41,
	0x64, 0x64, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x7d, 0x0a, 0x04, 0x72, 0x65, 0x70, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x43, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x72, 0x67,
	0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2e, 0x61, 0x72, 0x67, 0x6f, 0x63, 0x64, 0x5f, 0x61, 0x75, 0x74,
	0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x61, 0x70, 0x69, 0x73, 0x2e,
	0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x42, 0x24, 0x92, 0x41, 0x21, 0x32, 0x18, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x20, 0x74, 0x6f, 0x20, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x20, 0x6f, 0x6e, 0xd2, 0x01, 0x04, 0x72, 0x65, 0x70, 0x6f, 0x52, 0x04, 0x72, 0x65, 0x70, 0x6f,
	0x12, 0x77, 0x0a, 0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x41, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2e,
	0x61, 0x72, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2e, 0x61, 0x72, 0x67, 0x6f, 0x63, 0x64, 0x5f,
	0x61, 0x75, 0x74, 0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x61, 0x70,
	0x69, 0x73, 0x2e, 0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x61, 0x6e,
	0x69, 0x66, 0x65, 0x73, 0x74, 0x42, 0x18, 0x92, 0x41, 0x15, 0x32, 0x08, 0x6d, 0x61, 0x6e, 0x69,
	0x66, 0x65, 0x73, 0x74, 0xd2, 0x01, 0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52,
	0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x22, 0x4f, 0x0a, 0x13, 0x41, 0x64, 0x64,
	0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x38, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x24,
	0x92, 0x41, 0x21, 0x32, 0x1f, 0x77, 0x68, 0x65, 0x72, 0x65, 0x20, 0x74, 0x68, 0x65, 0x20, 0x6d,
	0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x20, 0x77, 0x61, 0x73, 0x20, 0x73, 0x61, 0x76, 0x65,
	0x64, 0x20, 0x74, 0x6f, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x32, 0xd1, 0x01, 0x0a, 0x06, 0x47,
	0x69, 0x74, 0x6f, 0x70, 0x73, 0x12, 0xc6, 0x01, 0x0a, 0x0b, 0x41, 0x64, 0x64, 0x4d, 0x61, 0x6e,
	0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x4b, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2e, 0x61, 0x72, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2e, 0x61, 0x72, 0x67, 0x6f,
	0x63, 0x64, 0x5f, 0x61, 0x75, 0x74, 0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2e, 0x70, 0x6b, 0x67,
	0x2e, 0x61, 0x70, 0x69, 0x73, 0x2e, 0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x76, 0x31, 0x2e,
	0x41, 0x64, 0x64, 0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x4c, 0x2e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2e,
	0x61, 0x72, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2e, 0x61, 0x72, 0x67, 0x6f, 0x63, 0x64, 0x5f,
	0x61, 0x75, 0x74, 0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2e, 0x70, 0x6b, 0x67, 0x2e, 0x61, 0x70,
	0x69, 0x73, 0x2e, 0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x64, 0x64,
	0x4d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x22, 0x11, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76,
	0x31, 0x2f, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x73, 0x3a, 0x01, 0x2a, 0x42, 0x39,
	0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x72, 0x67,
	0x6f, 0x70, 0x72, 0x6f, 0x6a, 0x2f, 0x61, 0x72, 0x67, 0x6f, 0x63, 0x64, 0x2d, 0x61, 0x75, 0x74,
	0x6f, 0x70, 0x69, 0x6c, 0x6f, 0x74, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f,
	0x67, 0x69, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_pkg_apis_gitops_v1_gitops_proto_rawDescOnce sync.Once
	file_pkg_apis_gitops_v1_gitops_proto_rawDescData = file_pkg_apis_gitops_v1_gitops_proto_rawDesc
)

func file_pkg_apis_gitops_v1_gitops_proto_rawDescGZIP() []byte {
	file_pkg_apis_gitops_v1_gitops_proto_rawDescOnce.Do(func() {
		file_pkg_apis_gitops_v1_gitops_proto_rawDescData = protoimpl.X.CompressGZIP(file_pkg_apis_gitops_v1_gitops_proto_rawDescData)
	})
	return file_pkg_apis_gitops_v1_gitops_proto_rawDescData
}

var file_pkg_apis_gitops_v1_gitops_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_pkg_apis_gitops_v1_gitops_proto_goTypes = []interface{}{
	(*Repository)(nil),          // 0: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Repository
	(*Manifest)(nil),            // 1: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Manifest
	(*AddManifestRequest)(nil),  // 2: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestRequest
	(*AddManifestResponse)(nil), // 3: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestResponse
}
var file_pkg_apis_gitops_v1_gitops_proto_depIdxs = []int32{
	0, // 0: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestRequest.repo:type_name -> github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Repository
	1, // 1: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestRequest.manifest:type_name -> github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Manifest
	2, // 2: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Gitops.AddManifest:input_type -> github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestRequest
	3, // 3: github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.Gitops.AddManifest:output_type -> github.com.argoproj.argocd_autopilot.pkg.apis.gitops.v1.AddManifestResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_pkg_apis_gitops_v1_gitops_proto_init() }
func file_pkg_apis_gitops_v1_gitops_proto_init() {
	if File_pkg_apis_gitops_v1_gitops_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_pkg_apis_gitops_v1_gitops_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Repository); i {
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
		file_pkg_apis_gitops_v1_gitops_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Manifest); i {
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
		file_pkg_apis_gitops_v1_gitops_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddManifestRequest); i {
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
		file_pkg_apis_gitops_v1_gitops_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddManifestResponse); i {
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
			RawDescriptor: file_pkg_apis_gitops_v1_gitops_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_apis_gitops_v1_gitops_proto_goTypes,
		DependencyIndexes: file_pkg_apis_gitops_v1_gitops_proto_depIdxs,
		MessageInfos:      file_pkg_apis_gitops_v1_gitops_proto_msgTypes,
	}.Build()
	File_pkg_apis_gitops_v1_gitops_proto = out.File
	file_pkg_apis_gitops_v1_gitops_proto_rawDesc = nil
	file_pkg_apis_gitops_v1_gitops_proto_goTypes = nil
	file_pkg_apis_gitops_v1_gitops_proto_depIdxs = nil
}