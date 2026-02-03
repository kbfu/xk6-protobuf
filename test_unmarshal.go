package main

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	// Create a simple message descriptor
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("test.proto"),
		Package: strPtr("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("TestMsg"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("field1"),
						Number: int32Ptr(1),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
					},
				},
			},
		},
	}
	
	files, _ := protodesc.NewFile(fileDesc, nil)
	msgDesc := files.Messages().ByName("TestMsg")
	msg := dynamicpb.NewMessage(msgDesc)
	
	// First unmarshal
	json1 := `{"field1": "value1_long_string_to_allocate_memory"}`
	protojson.Unmarshal([]byte(json1), msg)
	fmt.Printf("After 1st unmarshal: %v\n", msg)
	
	// Second unmarshal with shorter value - does it reuse buffer?
	json2 := `{"field1": "v2"}`
	protojson.Unmarshal([]byte(json2), msg)
	fmt.Printf("After 2nd unmarshal: %v\n", msg)
}

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type { return &t }
