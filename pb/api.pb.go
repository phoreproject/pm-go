// Code generated by protoc-gen-go.
// source: api.proto
// DO NOT EDIT!

/*
Package api is a generated protocol buffer package.

It is generated from these files:
	api.proto

It has these top-level messages:
	ListingReqApi
	ListingRespApi
	Inventory
	OrderRespApi
	CaseRespApi
	TransactionRecord
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type ListingReqApi struct {
	Listing   *Listing     `protobuf:"bytes,1,opt,name=listing" json:"listing,omitempty"`
	Inventory []*Inventory `protobuf:"bytes,2,rep,name=inventory" json:"inventory,omitempty"`
}

func (m *ListingReqApi) Reset()                    { *m = ListingReqApi{} }
func (m *ListingReqApi) String() string            { return proto.CompactTextString(m) }
func (*ListingReqApi) ProtoMessage()               {}
func (*ListingReqApi) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{0} }

func (m *ListingReqApi) GetListing() *Listing {
	if m != nil {
		return m.Listing
	}
	return nil
}

func (m *ListingReqApi) GetInventory() []*Inventory {
	if m != nil {
		return m.Inventory
	}
	return nil
}

type ListingRespApi struct {
	Contract  *RicardianContract `protobuf:"bytes,1,opt,name=contract" json:"contract,omitempty"`
	Inventory []*Inventory       `protobuf:"bytes,2,rep,name=inventory" json:"inventory,omitempty"`
}

func (m *ListingRespApi) Reset()                    { *m = ListingRespApi{} }
func (m *ListingRespApi) String() string            { return proto.CompactTextString(m) }
func (*ListingRespApi) ProtoMessage()               {}
func (*ListingRespApi) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{1} }

func (m *ListingRespApi) GetContract() *RicardianContract {
	if m != nil {
		return m.Contract
	}
	return nil
}

func (m *ListingRespApi) GetInventory() []*Inventory {
	if m != nil {
		return m.Inventory
	}
	return nil
}

type Inventory struct {
	Item  string `protobuf:"bytes,1,opt,name=item" json:"item,omitempty"`
	Count uint64 `protobuf:"varint,2,opt,name=count" json:"count,omitempty"`
}

func (m *Inventory) Reset()                    { *m = Inventory{} }
func (m *Inventory) String() string            { return proto.CompactTextString(m) }
func (*Inventory) ProtoMessage()               {}
func (*Inventory) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{2} }

type OrderRespApi struct {
	Contract     *RicardianContract   `protobuf:"bytes,1,opt,name=contract" json:"contract,omitempty"`
	State        OrderState           `protobuf:"varint,2,opt,name=state,enum=OrderState" json:"state,omitempty"`
	Read         bool                 `protobuf:"varint,3,opt,name=read" json:"read,omitempty"`
	Funded       bool                 `protobuf:"varint,4,opt,name=funded" json:"funded,omitempty"`
	Transactions []*TransactionRecord `protobuf:"bytes,5,rep,name=transactions" json:"transactions,omitempty"`
}

func (m *OrderRespApi) Reset()                    { *m = OrderRespApi{} }
func (m *OrderRespApi) String() string            { return proto.CompactTextString(m) }
func (*OrderRespApi) ProtoMessage()               {}
func (*OrderRespApi) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{3} }

func (m *OrderRespApi) GetContract() *RicardianContract {
	if m != nil {
		return m.Contract
	}
	return nil
}

func (m *OrderRespApi) GetTransactions() []*TransactionRecord {
	if m != nil {
		return m.Transactions
	}
	return nil
}

type CaseRespApi struct {
	BuyerContract            *RicardianContract           `protobuf:"bytes,1,opt,name=buyerContract" json:"buyerContract,omitempty"`
	VendorContract           *RicardianContract           `protobuf:"bytes,2,opt,name=vendorContract" json:"vendorContract,omitempty"`
	BuyerContractValidation  *CaseRespApi_ValidationError `protobuf:"bytes,3,opt,name=buyerContractValidation" json:"buyerContractValidation,omitempty"`
	VendorContractValidation *CaseRespApi_ValidationError `protobuf:"bytes,4,opt,name=vendorContractValidation" json:"vendorContractValidation,omitempty"`
	State                    OrderState                   `protobuf:"varint,5,opt,name=state,enum=OrderState" json:"state,omitempty"`
	Read                     bool                         `protobuf:"varint,6,opt,name=read" json:"read,omitempty"`
	BuyerOpened              bool                         `protobuf:"varint,7,opt,name=buyerOpened" json:"buyerOpened,omitempty"`
	Claim                    string                       `protobuf:"bytes,8,opt,name=claim" json:"claim,omitempty"`
}

func (m *CaseRespApi) Reset()                    { *m = CaseRespApi{} }
func (m *CaseRespApi) String() string            { return proto.CompactTextString(m) }
func (*CaseRespApi) ProtoMessage()               {}
func (*CaseRespApi) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{4} }

func (m *CaseRespApi) GetBuyerContract() *RicardianContract {
	if m != nil {
		return m.BuyerContract
	}
	return nil
}

func (m *CaseRespApi) GetVendorContract() *RicardianContract {
	if m != nil {
		return m.VendorContract
	}
	return nil
}

func (m *CaseRespApi) GetBuyerContractValidation() *CaseRespApi_ValidationError {
	if m != nil {
		return m.BuyerContractValidation
	}
	return nil
}

func (m *CaseRespApi) GetVendorContractValidation() *CaseRespApi_ValidationError {
	if m != nil {
		return m.VendorContractValidation
	}
	return nil
}

type CaseRespApi_ValidationError struct {
	Success bool     `protobuf:"varint,1,opt,name=success" json:"success,omitempty"`
	Errors  []string `protobuf:"bytes,2,rep,name=errors" json:"errors,omitempty"`
}

func (m *CaseRespApi_ValidationError) Reset()                    { *m = CaseRespApi_ValidationError{} }
func (m *CaseRespApi_ValidationError) String() string            { return proto.CompactTextString(m) }
func (*CaseRespApi_ValidationError) ProtoMessage()               {}
func (*CaseRespApi_ValidationError) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{4, 0} }

type TransactionRecord struct {
	Txid  string `protobuf:"bytes,1,opt,name=txid" json:"txid,omitempty"`
	Value int64  `protobuf:"varint,2,opt,name=value" json:"value,omitempty"`
}

func (m *TransactionRecord) Reset()                    { *m = TransactionRecord{} }
func (m *TransactionRecord) String() string            { return proto.CompactTextString(m) }
func (*TransactionRecord) ProtoMessage()               {}
func (*TransactionRecord) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{5} }

func init() {
	proto.RegisterType((*ListingReqApi)(nil), "ListingReqApi")
	proto.RegisterType((*ListingRespApi)(nil), "ListingRespApi")
	proto.RegisterType((*Inventory)(nil), "Inventory")
	proto.RegisterType((*OrderRespApi)(nil), "OrderRespApi")
	proto.RegisterType((*CaseRespApi)(nil), "CaseRespApi")
	proto.RegisterType((*CaseRespApi_ValidationError)(nil), "CaseRespApi.ValidationError")
	proto.RegisterType((*TransactionRecord)(nil), "TransactionRecord")
}

var fileDescriptor5 = []byte{
	// 453 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x9c, 0x53, 0x4d, 0x6f, 0xd4, 0x30,
	0x10, 0xd5, 0x76, 0xbf, 0x92, 0xc9, 0x76, 0x2b, 0x2c, 0x04, 0x56, 0xc5, 0x61, 0xc9, 0x69, 0x4f,
	0x3e, 0x2c, 0x02, 0x21, 0x24, 0x0e, 0x68, 0xc5, 0x01, 0x09, 0xa9, 0x92, 0x41, 0x15, 0x17, 0x0e,
	0x6e, 0x6c, 0x90, 0x51, 0x6a, 0x07, 0xdb, 0xa9, 0xe8, 0x0f, 0xe4, 0xc7, 0xf0, 0x2f, 0xb0, 0x9d,
	0x64, 0x93, 0x14, 0xad, 0x0a, 0xdc, 0x3c, 0x6f, 0xde, 0xbc, 0x79, 0x93, 0x99, 0x40, 0xca, 0x2a,
	0x49, 0x2a, 0xa3, 0x9d, 0x3e, 0x3f, 0x2b, 0xb4, 0x72, 0x86, 0x15, 0xce, 0xb6, 0xc0, 0x4a, 0x1b,
	0x2e, 0x4c, 0x1b, 0xe5, 0x9f, 0xe1, 0xf4, 0xbd, 0xb4, 0x4e, 0xaa, 0xaf, 0x54, 0x7c, 0x7f, 0x53,
	0x49, 0x94, 0xc3, 0xb2, 0x6c, 0x00, 0x3c, 0xd9, 0x4c, 0xb6, 0xd9, 0x2e, 0x21, 0x1d, 0xa1, 0x4b,
	0xa0, 0x2d, 0xa4, 0x52, 0xdd, 0x08, 0xe5, 0xb4, 0xb9, 0xc5, 0x27, 0x9b, 0xa9, 0x67, 0x01, 0x79,
	0xd7, 0x21, 0xb4, 0x4f, 0xe6, 0xdf, 0x60, 0x7d, 0x90, 0xb7, 0x55, 0xd0, 0x27, 0x90, 0x74, 0x8e,
	0xda, 0x06, 0x88, 0x50, 0x59, 0x30, 0xc3, 0x25, 0x53, 0xfb, 0x36, 0x43, 0x0f, 0x9c, 0x7f, 0xe8,
	0xf5, 0x1c, 0xd2, 0x03, 0x8e, 0x10, 0xcc, 0xa4, 0x13, 0xd7, 0xb1, 0x45, 0x4a, 0xe3, 0x1b, 0x3d,
	0x84, 0x79, 0xa1, 0x6b, 0xe5, 0xbc, 0xcc, 0x64, 0x3b, 0xa3, 0x4d, 0x90, 0xff, 0x9c, 0xc0, 0xea,
	0x22, 0x7c, 0x92, 0xff, 0x75, 0xf8, 0x14, 0xe6, 0xd6, 0x31, 0x27, 0xa2, 0xec, 0x7a, 0x97, 0x91,
	0xa8, 0xf6, 0x21, 0x40, 0xb4, 0xc9, 0x04, 0x37, 0x46, 0x30, 0x8e, 0xa7, 0x9e, 0x91, 0xd0, 0xf8,
	0x46, 0x8f, 0x60, 0xf1, 0xa5, 0x56, 0x5c, 0x70, 0x3c, 0x8b, 0x68, 0x1b, 0xa1, 0x17, 0xb0, 0xf2,
	0xba, 0xca, 0x7a, 0x69, 0xa9, 0x95, 0xc5, 0xf3, 0x38, 0x33, 0x22, 0x1f, 0x7b, 0x90, 0x8a, 0xc2,
	0x2f, 0x91, 0x8e, 0x78, 0xf9, 0xaf, 0x29, 0x64, 0x7b, 0x66, 0x45, 0x37, 0xc6, 0x4b, 0x38, 0xbd,
	0xaa, 0x6f, 0x85, 0xd9, 0xdf, 0x3f, 0xcb, 0x98, 0x88, 0x5e, 0xc1, 0xda, 0x7f, 0x46, 0xae, 0xfb,
	0xd2, 0x93, 0xa3, 0xa5, 0x77, 0x98, 0xe8, 0x12, 0x1e, 0x8f, 0xc4, 0x2e, 0x59, 0x29, 0x39, 0x0b,
	0x0e, 0xe3, 0xf0, 0xd9, 0xee, 0x09, 0x19, 0x98, 0x24, 0x7d, 0xfa, 0xad, 0x31, 0xda, 0xd0, 0x63,
	0xc5, 0xe8, 0x13, 0xe0, 0x71, 0xa7, 0x81, 0xf0, 0xec, 0x2f, 0x84, 0x8f, 0x56, 0xf7, 0xeb, 0x9b,
	0xdf, 0xbb, 0xbe, 0xc5, 0x60, 0x7d, 0x1b, 0xc8, 0xa2, 0xd7, 0x8b, 0x4a, 0x28, 0xbf, 0xc3, 0x65,
	0x4c, 0x0d, 0xa1, 0x78, 0x6e, 0x25, 0x93, 0xd7, 0x38, 0x89, 0x37, 0xd8, 0x04, 0xe7, 0x7b, 0x38,
	0xbb, 0xe3, 0x0d, 0x61, 0x58, 0xda, 0xba, 0x28, 0x84, 0xb5, 0x71, 0x47, 0x09, 0xed, 0xc2, 0x70,
	0x23, 0x22, 0x50, 0x6c, 0xbc, 0xfc, 0x94, 0xb6, 0x51, 0xfe, 0x1a, 0x1e, 0xfc, 0x71, 0x0e, 0xc1,
	0xa5, 0xfb, 0x21, 0x79, 0x77, 0xf2, 0xe1, 0x1d, 0x3c, 0xdc, 0xb0, 0xb2, 0x6e, 0x6e, 0x73, 0x4a,
	0x9b, 0xe0, 0x6a, 0x11, 0xff, 0xfd, 0x67, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0x7c, 0x82, 0x38,
	0xa6, 0x27, 0x04, 0x00, 0x00,
}
