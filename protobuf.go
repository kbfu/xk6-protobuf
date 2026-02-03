package protobuf

import (
	"context"
	"log"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/encoding/protojson"

	"go.k6.io/k6/js/modules"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func init() {
	modules.Register("k6/x/protobuf", new(Protobuf))
}

type Protobuf struct{}

type ProtoFile struct {
	messageDesc protoreflect.MessageDescriptor
	// Object pool to reuse dynamicpb.Message instances and reduce memory allocations
	messagePool sync.Pool
	// Byte buffer pool to reuse byte slices for marshal output
	bufferPool sync.Pool
}

func (p *Protobuf) Load(protoFilePath, lookupType string) ProtoFile {
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{},
	}

	files, err := compiler.Compile(context.Background(), protoFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if files == nil {
		log.Fatal("No files were passed as arguments")
	}
	if len(files) == 0 {
		log.Fatal("Zero files were parsed")
	}

	messageDesc := files[0].Messages().ByName(protoreflect.Name(lookupType))

	return ProtoFile{
		messageDesc: messageDesc,
		messagePool: sync.Pool{
			New: func() interface{} {
				return dynamicpb.NewMessage(messageDesc)
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate 1KB buffer (typical message size)
				b := make([]byte, 0, 1024)
				return &b
			},
		},
	}
}

func (p *ProtoFile) Encode(data string) []byte {
	// Get message from pool
	dynamicMessage := p.messagePool.Get().(*dynamicpb.Message)
	defer func() {
		// Reset message to clear all fields before returning to pool
		dynamicMessage.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			dynamicMessage.ProtoReflect().Clear(fd)
			return true
		})
		p.messagePool.Put(dynamicMessage)
	}()

	err := protojson.Unmarshal([]byte(data), dynamicMessage)

	if err != nil {
		log.Fatal(err)
	}

	// Get buffer from pool and use MarshalAppend to reuse buffer
	bufPtr := p.bufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // Reset length but keep capacity
	
	encodedBytes, err := proto.MarshalOptions{}.MarshalAppend(buf, dynamicMessage)
	if err != nil {
		log.Fatal(err)
	}

	// Make a copy to return (caller owns this)
	result := make([]byte, len(encodedBytes))
	copy(result, encodedBytes)
	
	// Return buffer to pool
	*bufPtr = buf
	p.bufferPool.Put(bufPtr)

	return result
}

func (p *ProtoFile) Decode(decodedBytes []byte) string {
	// Get message from pool
	decodedMessage := p.messagePool.Get().(*dynamicpb.Message)
	defer func() {
		// Reset message to clear all fields before returning to pool
		decodedMessage.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			decodedMessage.ProtoReflect().Clear(fd)
			return true
		})
		p.messagePool.Put(decodedMessage)
	}()

	err := proto.Unmarshal(decodedBytes, decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true,
	}

	jsonString, err := marshalOptions.Marshal(decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonString)
}
